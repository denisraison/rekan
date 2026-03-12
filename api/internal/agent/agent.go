package agent

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"

	baml "github.com/denisraison/rekan/api/internal/baml/baml_client"
	bamltypes "github.com/denisraison/rekan/api/internal/baml/baml_client/types"
	"github.com/pocketbase/pocketbase/core"
)

// Agent handles WhatsApp group messages for the operator chat.
type Agent struct {
	App       core.App
	WAClient  WAClient
	Logger    *slog.Logger
	Debouncer *Debouncer
}

// New creates a new Agent instance.
func New(app core.App, waClient WAClient, logger *slog.Logger) *Agent {
	return &Agent{
		App:       app,
		WAClient:  waClient,
		Logger:    logger,
		Debouncer: NewDebouncer(),
	}
}

// HandleGroupMessage is called for every incoming group message.
// Every message in the configured group is processed (group membership is the auth boundary).
func (a *Agent) HandleGroupMessage(evt *events.Message) {
	if evt.Message == nil {
		return
	}

	senderJID := evt.Info.Sender
	if senderJID.Server == types.HiddenUserServer {
		resolved := a.WAClient.ResolveLID(context.Background(), senderJID)
		if resolved.IsEmpty() {
			a.Logger.Warn("agent: skipping unresolvable LID", "sender", senderJID)
			return
		}
		senderJID = resolved
	}

	operatorName := evt.Info.PushName
	if operatorName == "" {
		operatorName = senderJID.User
	}
	operatorJID := senderJID.User

	text := extractText(evt)
	if text == "" {
		return
	}

	a.Debouncer.Submit(operatorJID, text, func(combined string) {
		a.processMessage(evt, combined, operatorName, operatorJID)
	})
}

func extractText(evt *events.Message) string {
	msg := evt.Message
	switch {
	case msg.GetConversation() != "":
		return msg.GetConversation()
	case msg.GetExtendedTextMessage() != nil:
		return msg.GetExtendedTextMessage().GetText()
	default:
		return ""
	}
}

func isConfirmation(msg string) bool {
	lower := strings.ToLower(strings.TrimSpace(msg))
	for _, w := range []string{"sim", "confirma", "isso", "pode fazer", "pode", "s"} {
		if lower == w {
			return true
		}
	}
	return false
}

func isCancellation(msg string) bool {
	lower := strings.ToLower(strings.TrimSpace(msg))
	for _, w := range []string{"não", "nao", "deixa", "cancela", "esquece", "n"} {
		if lower == w {
			return true
		}
	}
	return false
}

// bamlResult holds the output of a callBAML invocation.
type bamlResult struct {
	ReplyText    string
	ActionType   string
	ActionParams map[string]string
}

// callBAML calls the BAML agent and routes the action if present.
// Handles slow timer, error logging, and action routing.
func (a *Agent) callBAML(ctx context.Context, groupJID types.JID, app core.App, hydrated HydratedContext, state *OperatorState, operatorName, operatorJID, message string, start time.Time) (*bamlResult, error) {
	history, _ := LoadRecent(a.App, 15)
	historyText := FormatConversation(history)

	slowTimer := time.AfterFunc(5*time.Second, func() {
		SendReply(ctx, a.WAClient, groupJID, "Um momento...")
	})

	response, err := baml.AgentProcess(ctx, operatorName, message, hydrated.Formatted, historyText)
	slowTimer.Stop()

	if err != nil {
		a.Logger.Error("agent: BAML call failed", "error", err)
		LogAction(a.App, operatorName, operatorJID, "ERROR", nil, err.Error(), false, start)
		return nil, err
	}

	result := &bamlResult{ActionType: "INFO"}
	if response.Action != nil {
		result.ActionType = string(response.Action.ActionType)
		result.ActionParams = response.Action.ActionParams

		// If a new NEEDS_CONFIRMATION comes in while confirming, clear old state
		if state.State == "confirming" && response.Action.ActionStatus == bamltypes.AgentActionStatusNEEDS_CONFIRMATION {
			if err := ClearState(a.App, state, operatorJID); err != nil {
				a.Logger.Error("agent: failed to clear old state", "error", err)
			}
		}

		routeResult, routeErr := RouteAction(a.App, hydrated, state, *response.Action)
		if routeErr != nil {
			a.Logger.Error("agent: action routing failed", "error", routeErr, "action", result.ActionType)
			LogAction(a.App, operatorName, operatorJID, result.ActionType, result.ActionParams, routeErr.Error(), false, start)
			return nil, routeErr
		}
		if routeResult != "" {
			result.ReplyText = routeResult
		}
	}

	// Prefer BAML reply over router result
	if response.Reply != nil && *response.Reply != "" {
		result.ReplyText = *response.Reply
	}

	return result, nil
}

func (a *Agent) processMessage(evt *events.Message, message, operatorName, operatorJID string) {
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	groupJID := evt.Info.Chat

	if err := StoreMessage(a.App, operatorName, operatorJID, "user", message, ""); err != nil {
		a.Logger.Error("agent: failed to store message", "error", err)
	}

	state, _ := LoadState(a.App, operatorJID)

	// Handle confirmation/cancellation of pending actions
	if state.State == "confirming" {
		a.handleStatefulMessage(ctx, evt, message, operatorName, operatorJID, state, start)
		return
	}

	hydrated := HydrateContext(a.App, operatorName, operatorJID)

	ReactThumbsUp(ctx, a.WAClient, groupJID, string(evt.Info.ID), evt.Info.Sender)

	result, err := a.callBAML(ctx, groupJID, a.App, hydrated, state, operatorName, operatorJID, message, start)
	if err != nil {
		return
	}

	a.sendAndLog(ctx, groupJID, operatorName, operatorJID, result, start)
}

func (a *Agent) handleStatefulMessage(ctx context.Context, evt *events.Message, message, operatorName, operatorJID string, state *OperatorState, start time.Time) {
	groupJID := evt.Info.Chat
	actionType := state.ActionType

	var replyText string

	switch {
	case isConfirmation(message):
		// Only hydrate for execution (needs business records for update/pause)
		hydrated := HydrateContext(a.App, operatorName, operatorJID)
		result, err := ExecuteConfirmed(a.App, hydrated, state)
		if err != nil {
			a.Logger.Error("agent: confirmed action failed", "error", err, "action", actionType)
			replyText = operatorName + ", algo deu errado. Tenta de novo?"
			LogAction(a.App, operatorName, operatorJID, actionType, state.CollectedFields, err.Error(), false, start)
		} else {
			replyText = result
		}

	case isCancellation(message):
		if err := ClearState(a.App, state, operatorJID); err != nil {
			a.Logger.Error("agent: failed to clear state", "error", err)
		}
		replyText = operatorName + ", cancelado!"

	default:
		// Not confirmation or cancellation, pass to BAML
		hydrated := HydrateContext(a.App, operatorName, operatorJID)
		result, err := a.callBAML(ctx, groupJID, a.App, hydrated, state, operatorName, operatorJID, message, start)
		if err != nil {
			return
		}
		replyText = result.ReplyText
		actionType = result.ActionType
	}

	br := &bamlResult{ReplyText: replyText, ActionType: actionType, ActionParams: state.CollectedFields}

	ReactThumbsUp(ctx, a.WAClient, groupJID, string(evt.Info.ID), evt.Info.Sender)
	a.sendAndLog(ctx, groupJID, operatorName, operatorJID, br, start)
}

func (a *Agent) sendAndLog(ctx context.Context, groupJID types.JID, operatorName, operatorJID string, result *bamlResult, start time.Time) {
	if result.ReplyText == "" {
		LogAction(a.App, operatorName, operatorJID, result.ActionType, nil, "empty reply", true, start)
		return
	}

	if err := SendReply(ctx, a.WAClient, groupJID, result.ReplyText); err != nil {
		a.Logger.Error("agent: failed to send reply", "error", err)
		LogAction(a.App, operatorName, operatorJID, result.ActionType, nil, err.Error(), false, start)
		return
	}

	StoreMessage(a.App, "Rekan", "", "assistant", result.ReplyText, "")
	Prune(a.App, 15)
	LogAction(a.App, operatorName, operatorJID, result.ActionType, result.ActionParams, result.ReplyText, true, start)
}

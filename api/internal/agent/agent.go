package agent

import (
	"context"
	"log/slog"
	"slices"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"

	baml "github.com/denisraison/rekan/api/internal/baml/baml_client"
	bamltypes "github.com/denisraison/rekan/api/internal/baml/baml_client/types"
	content "github.com/denisraison/rekan/api/internal/content"
	"github.com/denisraison/rekan/api/internal/transcribe"
	"github.com/pocketbase/pocketbase/core"
)

// BAMLFunc is the signature for the BAML agent function, matching baml.AgentProcess.
type BAMLFunc func(ctx context.Context, operatorName, message, systemContext, conversationHistory string) (bamltypes.AgentResponse, error)

// Agent handles WhatsApp group messages for the operator chat.
type Agent struct {
	App        core.App
	WAClient   WAClient
	Logger     *slog.Logger
	Debouncer  *Debouncer
	Transcribe *transcribe.Client   // nil if GEMINI_API_KEY not set
	Generate   content.GenerateFunc // nil if not wired
	BAML       BAMLFunc             // defaults to baml.AgentProcess
}

// New creates a new Agent instance.
func New(app core.App, waClient WAClient, logger *slog.Logger, tc *transcribe.Client, gen content.GenerateFunc) *Agent {
	return &Agent{
		App:        app,
		WAClient:   waClient,
		Logger:     logger,
		Debouncer:  NewDebouncer(),
		Transcribe: tc,
		Generate:   gen,
		BAML: func(ctx context.Context, operatorName, message, systemContext, conversationHistory string) (bamltypes.AgentResponse, error) {
			return baml.AgentProcess(ctx, operatorName, message, systemContext, conversationHistory)
		},
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

	// Handle non-text media (images, audio, stickers, contacts, forwarded)
	if text == "" {
		media := ExtractMedia(context.Background(), a.WAClient, a.Transcribe, evt)
		if media.Text == "" && media.MediaType == "" {
			return
		}

		// Sticker with pending confirmation = "sim"
		if media.MediaType == "sticker" {
			state, _ := LoadState(a.App, operatorJID)
			if state.State == StateConfirming {
				text = "sim"
			} else {
				return
			}
		} else {
			text = media.Text
		}

		if text == "" {
			return
		}
	}

	groupJID := evt.Info.Chat
	messageID := string(evt.Info.ID)
	sender := senderJID

	a.Debouncer.Submit(operatorJID, text, func(combined string) {
		a.ProcessMessage(groupJID, messageID, sender, combined, operatorName, operatorJID)
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
	return slices.Contains([]string{"sim", "confirma", "isso", "pode fazer", "pode", "s"}, lower)
}

func isCancellation(msg string) bool {
	lower := strings.ToLower(strings.TrimSpace(msg))
	return slices.Contains([]string{"não", "nao", "deixa", "cancela", "esquece", "n"}, lower)
}

// bamlResult holds the output of a callBAML invocation.
type bamlResult struct {
	ReplyText    string
	ActionType   string
	ActionParams map[string]string
}

// callBAML calls the BAML agent and routes the action if present.
// Handles slow timer, error logging, and action routing.
func (a *Agent) callBAML(ctx context.Context, groupJID types.JID, hydrated HydratedContext, state *OperatorState, operatorName, operatorJID, message string, start time.Time) (*bamlResult, error) {
	history, _ := LoadRecentAndPrune(a.App, 15)
	historyText := FormatConversation(history)

	slowTimer := time.AfterFunc(5*time.Second, func() {
		SendReply(ctx, a.WAClient, groupJID, "Um momento...")
	})

	response, err := a.BAML(ctx, operatorName, message, hydrated.Formatted, historyText)
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
		if state.State == StateConfirming && response.Action.ActionStatus == bamltypes.AgentActionStatusNEEDS_CONFIRMATION {
			if err := ClearState(a.App, state, operatorJID); err != nil {
				a.Logger.Error("agent: failed to clear old state", "error", err)
			}
		}

		routeResult, routeErr := RouteAction(a.App, hydrated, state, *response.Action, a.Generate)
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

// ProcessMessage is the core message processing pipeline.
// Stores the message, loads conversation history, calls BAML, routes actions, and sends the reply.
// Used by HandleGroupMessage (via debouncer) and directly by tests.
func (a *Agent) ProcessMessage(groupJID types.JID, messageID string, senderJID types.JID, message, operatorName, operatorJID string) {
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := StoreMessage(a.App, operatorName, operatorJID, "user", message, ""); err != nil {
		a.Logger.Error("agent: failed to store message", "error", err)
	}

	state, _ := LoadState(a.App, operatorJID)

	// Handle confirmation/cancellation of pending actions
	if state.State == StateConfirming {
		a.handleStatefulMessage(ctx, groupJID, messageID, senderJID, message, operatorName, operatorJID, state, start)
		return
	}

	hydrated := HydrateContext(a.App, operatorName, operatorJID)

	ReactThumbsUp(ctx, a.WAClient, groupJID, messageID, senderJID)

	result, err := a.callBAML(ctx, groupJID, hydrated, state, operatorName, operatorJID, message, start)
	if err != nil {
		return
	}

	a.sendAndLog(ctx, groupJID, operatorName, operatorJID, result, start)
}

func (a *Agent) handleStatefulMessage(ctx context.Context, groupJID types.JID, messageID string, senderJID types.JID, message, operatorName, operatorJID string, state *OperatorState, start time.Time) {
	actionType := state.ActionType

	var replyText string

	switch {
	case isConfirmation(message):
		// Only hydrate for execution (needs business records for update/pause)
		hydrated := HydrateContext(a.App, operatorName, operatorJID)
		result, err := ExecuteConfirmed(a.App, hydrated, state, a.Generate)
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
		result, err := a.callBAML(ctx, groupJID, hydrated, state, operatorName, operatorJID, message, start)
		if err != nil {
			return
		}
		replyText = result.ReplyText
		actionType = result.ActionType
	}

	br := &bamlResult{ReplyText: replyText, ActionType: actionType, ActionParams: state.CollectedFields}

	ReactThumbsUp(ctx, a.WAClient, groupJID, messageID, senderJID)
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
	LogAction(a.App, operatorName, operatorJID, result.ActionType, result.ActionParams, result.ReplyText, true, start)
}

package agent

import (
	"context"
	"log/slog"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"

	baml "github.com/denisraison/rekan/api/internal/baml/baml_client"
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

	// Extract text content
	text := extractText(evt)
	if text == "" {
		return
	}

	// Submit to debouncer
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

func (a *Agent) processMessage(evt *events.Message, message, operatorName, operatorJID string) {
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	groupJID := evt.Info.Chat

	// Store incoming message in conversation buffer
	if err := StoreMessage(a.App, operatorName, operatorJID, "user", message, ""); err != nil {
		a.Logger.Error("agent: failed to store message", "error", err)
	}

	// Hydrate context and load conversation history
	hydrated := HydrateContext(a.App, operatorName)

	history, err := LoadRecent(a.App, 15)
	if err != nil {
		a.Logger.Error("agent: failed to load conversation", "error", err)
	}
	historyText := FormatConversation(history)

	// Send "Um momento..." if processing takes too long
	slowTimer := time.AfterFunc(5*time.Second, func() {
		waitMsg := "Um momento..."
		SendReply(ctx, a.WAClient, groupJID, waitMsg)
	})

	// Call BAML agent
	response, err := baml.AgentProcess(ctx, operatorName, message, hydrated.Formatted, historyText)
	slowTimer.Stop()

	if err != nil {
		a.Logger.Error("agent: BAML call failed", "error", err)
		LogAction(a.App, operatorName, operatorJID, "ERROR", nil, err.Error(), false, start)
		return
	}

	// React with thumbs up to indicate we're processing
	ReactThumbsUp(ctx, a.WAClient, groupJID, string(evt.Info.ID), evt.Info.Sender)

	// Route action if present
	var replyText string
	actionType := "INFO"
	if response.Action != nil {
		actionType = string(response.Action.ActionType)
		result, routeErr := RouteAction(hydrated, *response.Action)
		if routeErr != nil {
			a.Logger.Error("agent: action routing failed", "error", routeErr, "action", actionType)
			LogAction(a.App, operatorName, operatorJID, actionType, response.Action.ActionParams, routeErr.Error(), false, start)
			return
		}
		replyText = result
	}

	// Prefer BAML reply over router result if both exist
	if response.Reply != nil && *response.Reply != "" {
		replyText = *response.Reply
	}

	if replyText == "" {
		LogAction(a.App, operatorName, operatorJID, actionType, nil, "empty reply", true, start)
		return
	}

	// Send reply
	if err := SendReply(ctx, a.WAClient, groupJID, replyText); err != nil {
		a.Logger.Error("agent: failed to send reply", "error", err)
		LogAction(a.App, operatorName, operatorJID, actionType, nil, err.Error(), false, start)
		return
	}

	// Store agent reply in conversation buffer
	StoreMessage(a.App, "Rekan", "", "assistant", replyText, "")

	// Prune old messages
	Prune(a.App, 15)

	// Log successful action
	var actionParams map[string]string
	if response.Action != nil {
		actionParams = response.Action.ActionParams
	}
	LogAction(a.App, operatorName, operatorJID, actionType, actionParams, replyText, true, start)
}

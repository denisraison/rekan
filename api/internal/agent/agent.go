package agent

import (
	"context"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"

	content "github.com/denisraison/rekan/api/internal/content"
	"github.com/denisraison/rekan/api/internal/transcribe"
	wa "github.com/denisraison/rekan/api/internal/whatsapp"
	"github.com/pocketbase/pocketbase/core"
)

// Agent handles WhatsApp group messages for the operator chat.
type Agent struct {
	App        core.App
	WAClient   WAClient
	Logger     *slog.Logger
	Debouncer  *Debouncer
	Transcribe *transcribe.Client   // nil if GEMINI_API_KEY not set
	Generate   content.GenerateFunc // nil if not wired
	Claude     *ClaudeClient
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
		Claude:     NewClaudeClient(),
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
			state, err := LoadState(a.App, operatorJID)
			if err != nil {
				a.Logger.Error("agent: load state for sticker", "error", err)
				return
			}
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
	messageID := evt.Info.ID
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

// agentResult holds the output of the tool-use loop.
type agentResult struct {
	ReplyText  string
	ActionType string
}

// ProcessMessage is the core message processing pipeline.
// Stores the message, loads conversation history, uses tool-use loop, and sends the reply.
// Used by HandleGroupMessage (via debouncer) and directly by tests.
func (a *Agent) ProcessMessage(groupJID types.JID, messageID string, senderJID types.JID, message, operatorName, operatorJID string) {
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := StoreMessage(a.App, operatorName, operatorJID, "user", message, ""); err != nil {
		a.Logger.Error("agent: failed to store message", "error", err)
	}

	state, err := LoadState(a.App, operatorJID)
	if err != nil {
		a.Logger.Error("agent: load state", "error", err)
	}

	// Handle confirmation/cancellation of pending actions
	if state.State == StateConfirming {
		a.handleStatefulMessage(ctx, groupJID, messageID, senderJID, message, operatorName, operatorJID, state, start)
		return
	}

	if err := ReactThumbsUp(ctx, a.WAClient, groupJID, messageID, senderJID); err != nil {
		a.Logger.Error("agent: react thumbs up", "error", err)
	}

	stop := wa.Typing(ctx, a.WAClient, groupJID)
	defer stop()

	result, err := a.processWithTools(ctx, groupJID, state, operatorName, operatorJID, message)
	if err != nil {
		a.Logger.Error("agent: tool-use loop failed", "error", err)
		LogAction(a.App, operatorName, operatorJID, "ERROR", nil, err.Error(), false, start)
		if sendErr := SendReply(ctx, a.WAClient, groupJID, operatorName+", algo deu errado. Tenta de novo?"); sendErr != nil {
			a.Logger.Error("agent: failed to send error reply", "error", sendErr)
		}
		return
	}

	a.sendAndLog(ctx, groupJID, operatorName, operatorJID, result, start)
}

// processWithTools runs the Claude tool-use loop for a message.
func (a *Agent) processWithTools(ctx context.Context, groupJID types.JID, state *OperatorState, operatorName, operatorJID, message string) (*agentResult, error) {
	history, err := LoadRecentAndPrune(a.App, 15)
	if err != nil {
		a.Logger.Error("agent: failed to load conversation history", "error", err)
	}

	// Build Claude messages from conversation history
	messages := buildClaudeMessages(history, message)

	slowTimer := time.AfterFunc(5*time.Second, func() {
		if err := SendReply(ctx, a.WAClient, groupJID, "Um momento..."); err != nil {
			a.Logger.Error("agent: failed to send slow timer reply", "error", err)
		}
	})

	systemPrompt := buildSystemPrompt(operatorName)
	tuResult, err := a.Claude.RunToolLoop(ctx, a.App, state, operatorName, operatorJID, a.Generate, messages, systemPrompt)
	slowTimer.Stop()

	if err != nil {
		return nil, err
	}

	actionType := "INFO"
	if tuResult.WriteUsed && len(tuResult.ToolsCalled) > 0 {
		for i := len(tuResult.ToolsCalled) - 1; i >= 0; i-- {
			if at := toolNameToActionType(tuResult.ToolsCalled[i]); at != "" {
				actionType = at
				break
			}
		}
	} else if len(tuResult.ToolsCalled) > 0 {
		if at := toolNameToActionType(tuResult.ToolsCalled[0]); at != "" {
			actionType = at
		}
	}

	reply := tuResult.Reply
	if len(tuResult.Posts) > 0 {
		reply += "\n\n" + formatPostDetails(tuResult.BizNames, tuResult.Posts)
	}

	return &agentResult{ReplyText: reply, ActionType: actionType}, nil
}

// buildClaudeMessages converts conversation history + current message into Claude API messages.
func buildClaudeMessages(history []ConversationMessage, currentMessage string) []anthropic.MessageParam {
	var messages []anthropic.MessageParam

	for _, msg := range history {
		text := msg.Content
		if msg.Role == "user" {
			messages = append(messages, anthropic.NewUserMessage(anthropic.NewTextBlock(text)))
		} else {
			messages = append(messages, anthropic.MessageParam{
				Role:    anthropic.MessageParamRoleAssistant,
				Content: []anthropic.ContentBlockParamUnion{anthropic.NewTextBlock(text)},
			})
		}
	}

	// Ensure messages alternate roles (Claude API requirement)
	messages = mergeConsecutiveRoles(messages)

	// Add current message
	messages = append(messages, anthropic.NewUserMessage(anthropic.NewTextBlock(currentMessage)))

	return messages
}

// mergeConsecutiveRoles merges consecutive messages with the same role.
func mergeConsecutiveRoles(messages []anthropic.MessageParam) []anthropic.MessageParam {
	if len(messages) == 0 {
		return messages
	}

	var merged []anthropic.MessageParam
	for _, msg := range messages {
		if len(merged) > 0 && merged[len(merged)-1].Role == msg.Role {
			merged[len(merged)-1].Content = append(merged[len(merged)-1].Content, msg.Content...)
		} else {
			merged = append(merged, msg)
		}
	}

	// Claude requires first message to be user role
	if len(merged) > 0 && merged[0].Role != anthropic.MessageParamRoleUser {
		merged = merged[1:]
	}

	return merged
}

// toolNameToActionType maps tool names to action type strings for logging.
func toolNameToActionType(name string) string {
	switch name {
	case "find_customer":
		return "CUSTOMER_INFO"
	case "list_customers":
		return "CUSTOMER_LIST"
	case "find_post", "list_posts":
		return "POST_LIST_PENDING"
	case "recent_activity":
		return "STATUS_OVERVIEW"
	case "create_customer":
		return ActionCustomerCreate
	case "update_customer":
		return ActionCustomerUpdate
	case "pause_customer":
		return ActionCustomerPause
	case "generate_post":
		return ActionPostGenerate
	case "approve_post":
		return ActionPostApprove
	case "reject_post":
		return ActionPostReject
	default:
		return ""
	}
}

func (a *Agent) handleStatefulMessage(ctx context.Context, groupJID types.JID, messageID string, senderJID types.JID, message, operatorName, operatorJID string, state *OperatorState, start time.Time) {
	stop := wa.Typing(ctx, a.WAClient, groupJID)
	defer stop()

	actionType := state.ActionType

	var replyText string

	switch {
	case isConfirmation(message):
		result, err := ExecuteConfirmed(ctx, a.App, operatorName, state, a.Generate)
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
		// Not confirmation or cancellation, process normally via tool-use
		result, err := a.processWithTools(ctx, groupJID, state, operatorName, operatorJID, message)
		if err != nil {
			a.Logger.Error("agent: tool-use in stateful context failed", "error", err)
			return
		}
		replyText = result.ReplyText
		actionType = result.ActionType
	}

	br := &agentResult{ReplyText: replyText, ActionType: actionType}

	if err := ReactThumbsUp(ctx, a.WAClient, groupJID, messageID, senderJID); err != nil {
		a.Logger.Error("agent: react thumbs up", "error", err)
	}
	a.sendAndLog(ctx, groupJID, operatorName, operatorJID, br, start)
}

func (a *Agent) sendAndLog(ctx context.Context, groupJID types.JID, operatorName, operatorJID string, result *agentResult, start time.Time) {
	if result.ReplyText == "" {
		LogAction(a.App, operatorName, operatorJID, result.ActionType, nil, "empty reply", true, start)
		return
	}

	if err := SendReply(ctx, a.WAClient, groupJID, result.ReplyText); err != nil {
		a.Logger.Error("agent: failed to send reply", "error", err)
		LogAction(a.App, operatorName, operatorJID, result.ActionType, nil, err.Error(), false, start)
		return
	}

	if err := StoreMessage(a.App, "Rekan", "", "assistant", result.ReplyText, ""); err != nil {
		a.Logger.Error("agent: failed to store assistant message", "error", err)
	}
	LogAction(a.App, operatorName, operatorJID, result.ActionType, nil, result.ReplyText, true, start)
}

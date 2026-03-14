package agent

import (
	"context"
	"encoding/json"
	"fmt"
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

const fallbackWriteReply = ", anotado! Aguardando sua confirmação."

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

func IsConfirmation(msg string) bool {
	lower := strings.ToLower(strings.TrimSpace(msg))
	return slices.Contains([]string{"sim", "confirma", "isso", "pode fazer", "pode", "s"}, lower)
}

func IsCancellation(msg string) bool {
	lower := strings.ToLower(strings.TrimSpace(msg))
	return slices.Contains([]string{"não", "nao", "deixa", "cancela", "esquece", "n", "para"}, lower)
}

// agentResult holds the output of the tool-use loop.
type agentResult struct {
	ReplyText   string
	ToolSummary string
	ActionType  string
	LoopMsgs    []anthropic.MessageParam // tool loop messages for structured storage
	FinalMsg    anthropic.MessageParam   // actual final assistant response from Claude
}

// ProcessMessage is the core message processing pipeline.
// Stores the message, loads conversation history, uses tool-use loop, and sends the reply.
// Used by HandleGroupMessage (via debouncer) and directly by tests.
func (a *Agent) ProcessMessage(groupJID types.JID, messageID string, senderJID types.JID, message, operatorName, operatorJID string) {
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	userStructured := marshalMessageParam(anthropic.NewUserMessage(anthropic.NewTextBlock(message)))
	if err := StoreMessage(a.App, operatorName, operatorJID, "user", message, "", userStructured); err != nil {
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
	// If a write tool was called but Claude produced no text, use a fallback
	if reply == "" && tuResult.WriteUsed && len(tuResult.ToolsCalled) > 0 {
		reply = operatorName + fallbackWriteReply
	}
	if len(tuResult.Posts) > 0 {
		reply += "\n\n" + formatPostDetails(tuResult.BizNames, tuResult.Posts)
	}

	return &agentResult{
		ReplyText:   reply,
		ToolSummary: buildToolSummary(tuResult.ToolLog),
		ActionType:  actionType,
		LoopMsgs:    tuResult.LoopMsgs,
		FinalMsg:    tuResult.FinalMsg,
	}, nil
}

// buildClaudeMessages converts conversation history + current message into Claude API messages.
func buildClaudeMessages(history []ConversationMessage, currentMessage string) []anthropic.MessageParam {
	var messages []anthropic.MessageParam

	for _, msg := range history {
		// Prefer structured JSON when available (preserves tool_use/tool_result blocks)
		if msg.Structured != "" {
			var mp anthropic.MessageParam
			if json.Unmarshal([]byte(msg.Structured), &mp) == nil {
				messages = append(messages, mp)
				continue
			}
		}

		// Fallback: plain text for old messages without structured data
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

// describeAction builds a human-readable description of the pending action for the LLM classifier.
func describeAction(state *OperatorState) string {
	var nameHolder struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(state.CollectedFields, &nameHolder); err != nil {
		return state.ActionType
	}

	labels := map[string]string{
		ActionCustomerCreate: "Cadastrar cliente",
		ActionCustomerUpdate: "Alterar cliente",
		ActionCustomerPause:  "Pausar cliente",
		ActionPostGenerate:   "Gerar post",
		ActionPostApprove:    "Aprovar post",
		ActionPostReject:     "Rejeitar post",
	}

	label := labels[state.ActionType]
	if label == "" {
		label = state.ActionType
	}
	if nameHolder.Name != "" {
		return label + " " + nameHolder.Name
	}
	return label
}

func (a *Agent) handleStatefulMessage(ctx context.Context, groupJID types.JID, messageID string, senderJID types.JID, message, operatorName, operatorJID string, state *OperatorState, start time.Time) {
	stop := wa.Typing(ctx, a.WAClient, groupJID)
	defer stop()

	actionType := state.ActionType

	var replyText string

	// Fast path: hardcoded word lists
	confirmed := IsConfirmation(message)
	cancelled := !confirmed && IsCancellation(message)

	// LLM fallback for messages the word lists don't catch
	if !confirmed && !cancelled {
		desc := describeAction(state)
		cls, err := a.Claude.ClassifyConfirmation(ctx, message, desc)
		if err != nil {
			a.Logger.Error("agent: classify confirmation", "error", err)
			replyText = operatorName + ", não entendi. Pode confirmar ou cancelar?"
			br := &agentResult{ReplyText: replyText, ActionType: actionType}
			if reactErr := ReactThumbsUp(ctx, a.WAClient, groupJID, messageID, senderJID); reactErr != nil {
				a.Logger.Error("agent: react thumbs up", "error", reactErr)
			}
			a.sendAndLog(ctx, groupJID, operatorName, operatorJID, br, start)
			return
		}
		switch cls {
		case ClassConfirm:
			confirmed = true
		case ClassCancel:
			cancelled = true
		case ClassOther:
			// treated as neither confirmation nor cancellation
		}
	}

	switch {
	case confirmed:
		result, err := ExecuteConfirmed(ctx, a.App, operatorName, state, a.Generate)
		if err != nil {
			a.Logger.Error("agent: confirmed action failed", "error", err, "action", actionType)
			replyText = operatorName + ", algo deu errado. Tenta de novo?"
			LogAction(a.App, operatorName, operatorJID, actionType, state.CollectedFields, err.Error(), false, start)
		} else {
			replyText = result
		}

	case cancelled:
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

	// Store tool loop messages (assistant tool_use + user tool_result pairs)
	for _, msg := range result.LoopMsgs {
		role := "assistant"
		if msg.Role == anthropic.MessageParamRoleUser {
			role = "user"
		}
		structured := marshalMessageParam(msg)
		if err := StoreMessage(a.App, "Rekan", "", role, "", "", structured); err != nil {
			a.Logger.Error("agent: failed to store loop message", "error", err)
		}
	}

	// Store final assistant reply with structured data
	storedContent := result.ReplyText
	if result.ToolSummary != "" {
		storedContent += "\n\n" + result.ToolSummary
	}
	// Use the actual final message from Claude when available (preserves all content blocks)
	finalMsg := result.FinalMsg
	if len(finalMsg.Content) == 0 {
		finalMsg = anthropic.MessageParam{
			Role:    anthropic.MessageParamRoleAssistant,
			Content: []anthropic.ContentBlockParamUnion{anthropic.NewTextBlock(result.ReplyText)},
		}
	}
	replyStructured := marshalMessageParam(finalMsg)

	if err := StoreMessage(a.App, "Rekan", "", "assistant", storedContent, "", replyStructured); err != nil {
		a.Logger.Error("agent: failed to store assistant message", "error", err)
	}
	LogAction(a.App, operatorName, operatorJID, result.ActionType, nil, result.ReplyText, true, start)
}

// marshalMessageParam serializes a MessageParam to JSON for the structured field.
func marshalMessageParam(mp anthropic.MessageParam) string {
	data, err := json.Marshal(mp)
	if err != nil {
		return ""
	}
	return string(data)
}

// buildToolSummary formats tool calls into a bracketed summary for conversation history.
func buildToolSummary(log []toolCallEntry) string {
	if len(log) == 0 {
		return ""
	}
	var parts []string
	for _, e := range log {
		parts = append(parts, fmt.Sprintf("%s(%s) -> %s", e.Name, e.Args, e.Result))
	}
	return "[Ferramentas: " + strings.Join(parts, ", ") + "]"
}

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

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
	Claude     *Client
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
		Claude:     NewClient(),
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

		if media.MediaType == "sticker" {
			text = "[Sticker]"
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

// agentResult holds the output of the tool-use loop.
type agentResult struct {
	ReplyText   string
	ToolSummary string
	ActionType  string
	LoopMsgs    []Message // tool loop messages for structured storage
	FinalMsg    Message   // actual final assistant response from Claude
}

// ProcessMessage is the core message processing pipeline.
// Stores the message, loads conversation history, uses tool-use loop, and sends the reply.
// Used by HandleGroupMessage (via debouncer) and directly by tests.
func (a *Agent) ProcessMessage(groupJID types.JID, messageID string, senderJID types.JID, message, operatorName, operatorJID string) {
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	userStructured := marshalMessage(NewUserMessage(NewTextBlock(message)))
	if err := StoreMessage(a.App, operatorName, operatorJID, "user", message, "", userStructured); err != nil {
		a.Logger.Error("agent: failed to store message", "error", err)
	}

	if err := ReactThumbsUp(ctx, a.WAClient, groupJID, messageID, senderJID); err != nil {
		a.Logger.Error("agent: react thumbs up", "error", err)
	}

	stop := wa.Typing(ctx, a.WAClient, groupJID)
	defer stop()

	result, err := a.processWithTools(ctx, groupJID, operatorName, message)
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
func (a *Agent) processWithTools(ctx context.Context, groupJID types.JID, operatorName, message string) (*agentResult, error) {
	history, err := LoadRecentAndPrune(a.App, 15)
	if err != nil {
		a.Logger.Error("agent: failed to load conversation history", "error", err)
	}

	messages := buildClaudeMessages(history, message)

	executor := &ToolExecutor{
		Ctx:      ctx,
		App:      a.App,
		WAClient: a.WAClient,
		Generate: a.Generate,
	}
	tools := buildTools(executor, operatorName)

	slowTimer := time.AfterFunc(5*time.Second, func() {
		if err := SendReply(ctx, a.WAClient, groupJID, "Um momento..."); err != nil {
			a.Logger.Error("agent: failed to send slow timer reply", "error", err)
		}
		wa.Typing(ctx, a.WAClient, groupJID)
	})

	systemPrompt := buildSystemPrompt(operatorName)
	runResult, runErr := a.Claude.Run(ctx, RunConfig{
		System:   systemPrompt,
		Messages: messages,
		Tools:    tools,
		MaxTurns: maxToolRoundTrips,
	})
	slowTimer.Stop()

	if runErr != nil {
		return nil, runErr
	}

	// Extract tool calls from traces for action type mapping and tool log
	var toolsCalled []string
	var toolLog []toolCallEntry
	for _, trace := range runResult.Traces {
		for _, tc := range trace.ToolCalls {
			toolsCalled = append(toolsCalled, tc.Name)
			toolLog = append(toolLog, toolCallEntry{Name: tc.Name})
		}
	}

	actionType := "INFO"
	if executor.WriteUsed && len(toolsCalled) > 0 {
		for i := len(toolsCalled) - 1; i >= 0; i-- {
			if at := toolNameToActionType(toolsCalled[i]); at != "" {
				actionType = at
				break
			}
		}
	} else if len(toolsCalled) > 0 {
		if at := toolNameToActionType(toolsCalled[0]); at != "" {
			actionType = at
		}
	}

	reply := runResult.Reply

	// Collect loop messages (all except the final) and the final assistant message
	var loopMsgs []Message
	var finalMsg Message
	allMsgs := runResult.Messages
	if len(allMsgs) > len(messages) {
		newMsgs := allMsgs[len(messages):]
		if len(newMsgs) > 0 {
			finalMsg = newMsgs[len(newMsgs)-1]
			loopMsgs = newMsgs[:len(newMsgs)-1]
		}
	}

	return &agentResult{
		ReplyText:   reply,
		ToolSummary: buildToolSummary(toolLog),
		ActionType:  actionType,
		LoopMsgs:    loopMsgs,
		FinalMsg:    finalMsg,
	}, nil
}

const maxToolRoundTrips = 5

// buildClaudeMessages converts conversation history + current message into agent messages.
func buildClaudeMessages(history []ConversationMessage, currentMessage string) []Message {
	var messages []Message

	for _, msg := range history {
		// Prefer structured JSON when available (preserves tool_use/tool_result blocks)
		if msg.Structured != "" {
			var m Message
			if json.Unmarshal([]byte(msg.Structured), &m) == nil {
				messages = append(messages, m)
				continue
			}
		}

		// Fallback: plain text for old messages without structured data.
		// Skip empty content (e.g. tool loop messages whose structured JSON
		// failed to deserialize due to content type mismatch).
		if msg.Content == "" {
			continue
		}
		if msg.Role == "user" {
			messages = append(messages, NewUserMessage(NewTextBlock(msg.Content)))
		} else {
			messages = append(messages, NewAssistantMessage(NewTextBlock(msg.Content)))
		}
	}

	// Ensure messages alternate roles (Claude API requirement)
	messages = mergeConsecutiveRoles(messages)

	// Remove unpaired tool_use/tool_result blocks (e.g. from history pruning)
	// then re-merge since stripping can create new consecutive same-role messages
	messages = mergeConsecutiveRoles(sanitizeToolPairs(messages))

	// Add current message (may already be in history since ProcessMessage stores it before loading)
	messages = appendOrMergeUser(messages, currentMessage)

	return messages
}

// sanitizeToolPairs strips unpaired tool_use and tool_result blocks.
func sanitizeToolPairs(messages []Message) []Message {
	paired := map[string]bool{}
	for i, msg := range messages {
		if msg.Role != RoleAssistant || i+1 >= len(messages) {
			continue
		}
		next := messages[i+1]
		if next.Role != RoleUser {
			continue
		}
		resultIDs := map[string]bool{}
		for _, block := range next.Content {
			if block.Type == "tool_result" {
				resultIDs[block.ToolUseID] = true
			}
		}
		for _, block := range msg.Content {
			if block.Type == "tool_use" && resultIDs[block.ID] {
				paired[block.ID] = true
			}
		}
	}

	var result []Message
	for _, msg := range messages {
		var kept []ContentBlock
		for _, block := range msg.Content {
			switch {
			case block.Type == "tool_use" && !paired[block.ID]:
				continue
			case block.Type == "tool_result" && !paired[block.ToolUseID]:
				continue
			}
			kept = append(kept, block)
		}
		if len(kept) > 0 {
			result = append(result, Message{Role: msg.Role, Content: kept})
		}
	}
	return result
}

// appendOrMergeUser adds the current message. If the last message is already
// a user message containing the same text, it skips the duplicate.
func appendOrMergeUser(messages []Message, text string) []Message {
	if n := len(messages); n > 0 && messages[n-1].Role == RoleUser {
		for _, block := range messages[n-1].Content {
			if block.Type == "text" && block.Text == text {
				return messages
			}
		}
	}
	return append(messages, NewUserMessage(NewTextBlock(text)))
}

// mergeConsecutiveRoles merges consecutive messages with the same role.
func mergeConsecutiveRoles(messages []Message) []Message {
	if len(messages) == 0 {
		return messages
	}

	var merged []Message
	for _, msg := range messages {
		if len(merged) > 0 && merged[len(merged)-1].Role == msg.Role {
			merged[len(merged)-1].Content = append(merged[len(merged)-1].Content, msg.Content...)
		} else {
			merged = append(merged, msg)
		}
	}

	// Claude requires first message to be user role
	if len(merged) > 0 && merged[0].Role != RoleUser {
		merged = merged[1:]
	}

	return merged
}

// toolNameToActionType maps tool names to action type strings for logging.
func toolNameToActionType(name string) string {
	switch name {
	case "search_customers":
		return "CUSTOMER_INFO"
	case "search_posts":
		return "POST_LIST_PENDING"
	case "create_customer":
		return ActionCustomerCreate
	case "update_customer":
		return ActionCustomerUpdate
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
		structured := marshalMessage(msg)
		if err := StoreMessage(a.App, "Rekan", "", string(msg.Role), "", "", structured); err != nil {
			a.Logger.Error("agent: failed to store loop message", "error", err)
		}
	}

	// Store final assistant reply with structured data
	storedContent := result.ReplyText
	if result.ToolSummary != "" {
		storedContent += "\n\n" + result.ToolSummary
	}
	finalMsg := result.FinalMsg
	if len(finalMsg.Content) == 0 {
		finalMsg = NewAssistantMessage(NewTextBlock(result.ReplyText))
	}
	replyStructured := marshalMessage(finalMsg)
	if err := StoreMessage(a.App, "Rekan", "", "assistant", storedContent, "", replyStructured); err != nil {
		a.Logger.Error("agent: failed to store assistant message", "error", err)
	}
	LogAction(a.App, operatorName, operatorJID, result.ActionType, nil, result.ReplyText, true, start)
}

// marshalMessage serializes a Message to JSON for the structured field.
func marshalMessage(m Message) string {
	data, err := json.Marshal(m)
	if err != nil {
		return ""
	}
	return string(data)
}

// toolCallEntry records a single tool call for building summaries.
type toolCallEntry struct {
	Name   string
	Args   string // abbreviated args
	Result string // abbreviated result
}

// buildToolSummary formats tool calls into a bracketed summary for conversation history.
func buildToolSummary(log []toolCallEntry) string {
	if len(log) == 0 {
		return ""
	}
	var parts []string
	for _, e := range log {
		if e.Args != "" || e.Result != "" {
			parts = append(parts, fmt.Sprintf("%s(%s) -> %s", e.Name, e.Args, e.Result))
		} else {
			parts = append(parts, e.Name)
		}
	}
	return "[Ferramentas: " + strings.Join(parts, ", ") + "]"
}

package agent

import (
	"context"
	"encoding/json"
)

// Role is a conversation participant.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// Message is a conversation turn.
type Message struct {
	Role    Role           `json:"role"`
	Content []ContentBlock `json:"content"`
}

// ContentBlock is one piece of content within a message.
type ContentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   string          `json:"content,omitempty"`
	IsError   bool            `json:"is_error,omitempty"`
}

// Tool is the unit of composition. Everything the agent can do is a Tool.
type Tool struct {
	Name        string
	Description string
	InputSchema json.RawMessage
	Execute     func(ctx context.Context, input json.RawMessage) (string, error)
}

// Trace records one turn of the agent loop.
type Trace struct {
	Turn         int         `json:"turn"`
	ModelLatency int64       `json:"model_latency_ms"`
	InputTokens  int         `json:"input_tokens"`
	OutputTokens int         `json:"output_tokens"`
	ToolCalls    []ToolTrace `json:"tool_calls,omitempty"`
}

// ToolTrace records a single tool execution within a turn.
type ToolTrace struct {
	Name     string `json:"name"`
	Duration int64  `json:"duration_ms"`
	Error    string `json:"error,omitempty"`
}

// RunConfig configures a single agent loop run.
type RunConfig struct {
	System    string
	Messages  []Message
	Tools     []Tool
	MaxTurns  int         // default 10
	MaxTokens int         // default 2048
	OnTrace   func(Trace) // optional
}

// RunResult is the output of Run().
type RunResult struct {
	Reply    string
	Messages []Message // full conversation including tool turns
	Traces   []Trace   // one per turn
}

// NewTextBlock creates a text content block.
func NewTextBlock(text string) ContentBlock {
	return ContentBlock{Type: "text", Text: text}
}

// NewToolResultBlock creates a tool_result content block.
func NewToolResultBlock(toolUseID, content string, isError bool) ContentBlock {
	return ContentBlock{
		Type:      "tool_result",
		ToolUseID: toolUseID,
		Content:   content,
		IsError:   isError,
	}
}

// NewUserMessage creates a user message with the given content blocks.
func NewUserMessage(blocks ...ContentBlock) Message {
	return Message{Role: RoleUser, Content: blocks}
}

// NewAssistantMessage creates an assistant message with the given content blocks.
func NewAssistantMessage(blocks ...ContentBlock) Message {
	return Message{Role: RoleAssistant, Content: blocks}
}

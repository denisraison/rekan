package agent

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	content "github.com/denisraison/rekan/api/internal/content"
	"github.com/pocketbase/pocketbase/core"
)

const maxToolRoundTrips = 5

// ConfirmationClass is the result of classifying a message as confirmation, cancellation, or other.
type ConfirmationClass int

const (
	ClassOther  ConfirmationClass = iota
	ClassConfirm
	ClassCancel
)

// ClaudeClient wraps the Anthropic API for tool-use agent calls.
type ClaudeClient struct {
	client anthropic.Client
	model  anthropic.Model
}

// NewClaudeClient creates a client using CLAUDE_API_KEY from env.
func NewClaudeClient() *ClaudeClient {
	apiKey := os.Getenv("CLAUDE_API_KEY")
	return &ClaudeClient{
		client: anthropic.NewClient(option.WithAPIKey(apiKey)),
		model:  anthropic.ModelClaudeSonnet4_6,
	}
}

// toolCallEntry records a single tool call for building summaries.
type toolCallEntry struct {
	Name   string
	Args   string // abbreviated args
	Result string // abbreviated result
}

// toolUseResult is the output of a tool-use loop iteration.
type toolUseResult struct {
	Reply       string
	ToolsCalled []string
	ToolLog     []toolCallEntry
	WriteUsed   bool
	LoopMsgs    []anthropic.MessageParam // intermediate messages (tool_use + tool_result) for structured storage
	FinalMsg    anthropic.MessageParam   // the actual final assistant response from Claude
	Posts       []*core.Record           // posts referenced during execution, appended to reply
	BizNames    map[string]string        // business ID -> display name
}

// RunToolLoop runs the Claude tool-use loop until a final reply or max round trips.
func (cc *ClaudeClient) RunToolLoop(ctx context.Context, app core.App, state *OperatorState, operatorName, operatorJID string, gen content.GenerateFunc, messages []anthropic.MessageParam, systemPrompt string) (result *toolUseResult, err error) {
	tools := agentTools

	executor := &ToolExecutor{
		App:         app,
		State:       state,
		OperatorJID: operatorJID,
		Generate:    gen,
	}

	result = &toolUseResult{}
	defer func() {
		if err == nil {
			result.Posts = executor.Posts
			result.BizNames = executor.bizNameMap()
		}
	}()

	for range maxToolRoundTrips {
		var resp *anthropic.Message
		resp, err = cc.client.Messages.New(ctx, anthropic.MessageNewParams{
			Model:     cc.model,
			MaxTokens: 1024,
			System:    []anthropic.TextBlockParam{{Text: systemPrompt}},
			Messages:  messages,
			Tools:     tools,
		})
		if err != nil {
			return nil, fmt.Errorf("claude API: %w", err)
		}

		// Add assistant response to messages
		assistantMsg := resp.ToParam()
		messages = append(messages, assistantMsg)

		// Collect tool calls and text
		var toolResults []anthropic.ContentBlockParamUnion
		for _, block := range resp.Content {
			switch v := block.AsAny().(type) {
			case anthropic.TextBlock:
				result.Reply = v.Text
			case anthropic.ToolUseBlock:
				tr := executor.executeTool(v.Name, v.Input, operatorName)
				result.ToolsCalled = append(result.ToolsCalled, v.Name)
				result.ToolLog = append(result.ToolLog, toolCallEntry{
					Name:   v.Name,
					Args:   truncate(string(v.Input), 80),
					Result: truncate(tr.Text, 60),
				})
				if tr.IsWrite {
					result.WriteUsed = true
				}
				toolResults = append(toolResults, anthropic.NewToolResultBlock(v.ID, tr.Text, false))
			}
		}

		// No tool calls means we're done
		if len(toolResults) == 0 {
			result.FinalMsg = assistantMsg
			return result, nil
		}

		// Capture intermediate messages for structured storage
		result.LoopMsgs = append(result.LoopMsgs, assistantMsg)
		toolResultMsg := anthropic.NewUserMessage(toolResults...)
		result.LoopMsgs = append(result.LoopMsgs, toolResultMsg)

		// If a write tool was called, stop the loop (wait for confirmation)
		if result.WriteUsed {
			return result, nil
		}

		// Feed tool results back
		messages = append(messages, toolResultMsg)
	}

	// Max round trips reached
	if result.Reply == "" {
		result.Reply = operatorName + ", não consegui processar. Tenta de novo?"
	}
	return result, nil
}

// ClassifyConfirmation uses Haiku to classify a message as confirmation, cancellation, or other.
// Only called as fallback when the hardcoded word lists don't match.
func (cc *ClaudeClient) ClassifyConfirmation(ctx context.Context, message, actionDescription string) (ConfirmationClass, error) {
	prompt := fmt.Sprintf(`A operadora tem uma ação pendente: "%s"
Ela respondeu: "%s"
Classifique: CONFIRMA, CANCELA, ou OUTRO
Responda apenas uma palavra.`, actionDescription, message)

	resp, err := cc.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeHaiku4_5,
		MaxTokens: 16,
		System:    []anthropic.TextBlockParam{{Text: "Responda apenas uma palavra: CONFIRMA, CANCELA, ou OUTRO."}},
		Messages:  []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock(prompt))},
	})
	if err != nil {
		return ClassOther, fmt.Errorf("classify confirmation: %w", err)
	}

	for _, block := range resp.Content {
		if tb, ok := block.AsAny().(anthropic.TextBlock); ok {
			word := strings.ToUpper(strings.TrimSpace(tb.Text))
			switch {
			case strings.HasPrefix(word, "CONFIRMA"):
				return ClassConfirm, nil
			case strings.HasPrefix(word, "CANCELA"):
				return ClassCancel, nil
			default:
				return ClassOther, nil
			}
		}
	}
	return ClassOther, nil
}

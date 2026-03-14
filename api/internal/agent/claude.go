package agent

import (
	"context"
	"fmt"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	content "github.com/denisraison/rekan/api/internal/content"
	"github.com/pocketbase/pocketbase/core"
)

const maxToolRoundTrips = 5

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
	PreviewUsed bool // true when a write tool returned a preview (confirmed=false)
	LoopMsgs    []anthropic.MessageParam // intermediate messages (tool_use + tool_result) for structured storage
	FinalMsg    anthropic.MessageParam   // the actual final assistant response from Claude
	Posts       []*core.Record           // posts referenced during execution, appended to reply
	BizNames    map[string]string        // business ID -> display name
}

// RunToolLoop runs the Claude tool-use loop until a final reply or max round trips.
func (cc *ClaudeClient) RunToolLoop(ctx context.Context, app core.App, operatorName string, gen content.GenerateFunc, messages []anthropic.MessageParam, systemPrompt string) (result *toolUseResult, err error) {
	tools := agentTools

	executor := &ToolExecutor{
		Ctx:      ctx,
		App:      app,
		Generate: gen,
	}

	result = &toolUseResult{}
	defer func() {
		if err == nil {
			result.Posts = executor.Posts
			if len(executor.Posts) > 0 {
				result.BizNames = executor.bizNameMap()
			}
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
		previewInRound := false
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
				if tr.IsPreview {
					previewInRound = true
					result.PreviewUsed = true
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

		// If a preview was returned, stop the loop so Claude can ask for confirmation
		if previewInRound {
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

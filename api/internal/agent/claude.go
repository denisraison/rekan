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

// toolUseResult is the output of a tool-use loop iteration.
type toolUseResult struct {
	Reply       string
	ToolsCalled []string
	WriteUsed   bool
}

// RunToolLoop runs the Claude tool-use loop until a final reply or max round trips.
func (cc *ClaudeClient) RunToolLoop(ctx context.Context, app core.App, state *OperatorState, operatorName, operatorJID string, gen content.GenerateFunc, messages []anthropic.MessageParam, systemPrompt string) (*toolUseResult, error) {
	tools := agentTools

	executor := &ToolExecutor{
		App:         app,
		State:       state,
		OperatorJID: operatorJID,
		Generate:    gen,
	}

	result := &toolUseResult{}

	for range maxToolRoundTrips {
		resp, err := cc.client.Messages.New(ctx, anthropic.MessageNewParams{
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
		messages = append(messages, resp.ToParam())

		// Collect tool calls and text
		var toolResults []anthropic.ContentBlockParamUnion
		for _, block := range resp.Content {
			switch v := block.AsAny().(type) {
			case anthropic.TextBlock:
				result.Reply = v.Text
			case anthropic.ToolUseBlock:
				tr := executor.executeTool(v.Name, v.Input, operatorName)
				result.ToolsCalled = append(result.ToolsCalled, v.Name)
				if tr.IsWrite {
					result.WriteUsed = true
				}
				toolResults = append(toolResults, anthropic.NewToolResultBlock(v.ID, tr.Text, false))
			}
		}

		// No tool calls means we're done
		if len(toolResults) == 0 {
			return result, nil
		}

		// If a write tool was called, stop the loop (wait for confirmation)
		if result.WriteUsed {
			return result, nil
		}

		// Feed tool results back
		messages = append(messages, anthropic.NewUserMessage(toolResults...))
	}

	// Max round trips reached
	if result.Reply == "" {
		result.Reply = operatorName + ", não consegui processar. Tenta de novo?"
	}
	return result, nil
}

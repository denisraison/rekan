package agent

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

const (
	defaultMaxTurns  = 10
	defaultMaxTokens = 2048
)

// Run executes the agent tool loop.
func (c *Client) Run(ctx context.Context, cfg RunConfig) (*RunResult, error) {
	maxTurns := cfg.MaxTurns
	if maxTurns <= 0 {
		maxTurns = defaultMaxTurns
	}
	maxTokens := cfg.MaxTokens
	if maxTokens <= 0 {
		maxTokens = defaultMaxTokens
	}

	messages := make([]Message, len(cfg.Messages))
	copy(messages, cfg.Messages)

	// Build tool lookup and API tool defs once (they don't change across turns)
	toolMap := make(map[string]Tool, len(cfg.Tools))
	var toolDefs []apiToolDef
	for _, t := range cfg.Tools {
		toolMap[t.Name] = t
		toolDefs = append(toolDefs, apiToolDef{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.InputSchema,
		})
	}

	result := &RunResult{}

	for turn := range maxTurns {
		callStart := time.Now()
		resp, err := c.call(ctx, cfg.System, messages, toolDefs, maxTokens)
		if err != nil {
			return nil, err
		}
		modelLatency := time.Since(callStart).Milliseconds()

		// Build assistant message from response
		assistantMsg := Message{Role: RoleAssistant, Content: resp.Content}
		messages = append(messages, assistantMsg)

		// Collect tool calls and text
		var toolUseBlocks []ContentBlock
		for _, block := range resp.Content {
			if block.Type == "text" {
				result.Reply = block.Text
			}
			if block.Type == "tool_use" {
				toolUseBlocks = append(toolUseBlocks, block)
			}
		}

		trace := Trace{
			Turn:         turn + 1,
			ModelLatency: modelLatency,
			InputTokens:  resp.Usage.InputTokens,
			OutputTokens: resp.Usage.OutputTokens,
		}

		// No tool calls means we're done
		if len(toolUseBlocks) == 0 {
			result.Traces = append(result.Traces, trace)
			if cfg.OnTrace != nil {
				cfg.OnTrace(trace)
			}
			result.Messages = messages
			return result, nil
		}

		// Execute tools concurrently
		type toolOutput struct {
			block    ContentBlock
			toolName string
			duration int64
			errMsg   string
		}
		outputs := make([]toolOutput, len(toolUseBlocks))

		var wg sync.WaitGroup
		for i, block := range toolUseBlocks {
			wg.Go(func() {
				toolStart := time.Now()

				t, ok := toolMap[block.Name]
				if !ok {
					outputs[i] = toolOutput{
						block:    NewToolResultBlock(block.ID, "unknown tool: "+block.Name, true),
						toolName: block.Name,
						duration: time.Since(toolStart).Milliseconds(),
						errMsg:   "unknown tool",
					}
					return
				}

				resultText, execErr := t.Execute(ctx, block.Input)
				dur := time.Since(toolStart).Milliseconds()

				isErr := execErr != nil
				content := resultText
				var errStr string
				if execErr != nil {
					content = execErr.Error()
					errStr = execErr.Error()
				}

				outputs[i] = toolOutput{
					block:    NewToolResultBlock(block.ID, content, isErr),
					toolName: block.Name,
					duration: dur,
					errMsg:   errStr,
				}
			})
		}
		wg.Wait()

		// Build tool result message and trace
		var resultBlocks []ContentBlock
		for _, o := range outputs {
			resultBlocks = append(resultBlocks, o.block)
			trace.ToolCalls = append(trace.ToolCalls, ToolTrace{
				Name:     o.toolName,
				Duration: o.duration,
				Error:    o.errMsg,
			})
		}

		result.Traces = append(result.Traces, trace)
		if cfg.OnTrace != nil {
			cfg.OnTrace(trace)
		}

		messages = append(messages, NewUserMessage(resultBlocks...))
	}

	// Max turns reached
	result.Messages = messages
	return result, nil
}

// marshalSchema converts a map to json.RawMessage for tool input schemas.
func marshalSchema(v any) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic("marshalSchema: " + err.Error())
	}
	return data
}

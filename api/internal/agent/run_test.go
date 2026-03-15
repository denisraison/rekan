package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// mockAPIResponse builds a JSON response matching the Anthropic Messages API format.
func mockAPIResponse(content []ContentBlock, stopReason string) []byte {
	resp := apiResponse{
		Content:    content,
		StopReason: stopReason,
		Usage:      apiUsage{InputTokens: 100, OutputTokens: 50},
	}
	data, _ := json.Marshal(resp)
	return data
}

func testClient(url string) *Client {
	return &Client{
		APIKey:  "test-key",
		Model:   "test-model",
		BaseURL: url,
	}
}

func TestRun_NoTools(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(mockAPIResponse([]ContentBlock{NewTextBlock("Oi!")}, "end_turn"))
	}))
	defer server.Close()

	client := testClient(server.URL)
	result, err := client.Run(context.Background(), RunConfig{
		System:   "You are helpful.",
		Messages: []Message{NewUserMessage(NewTextBlock("Oi"))},
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.Reply != "Oi!" {
		t.Errorf("reply: got %q, want %q", result.Reply, "Oi!")
	}
	if len(result.Traces) != 1 {
		t.Fatalf("traces: got %d, want 1", len(result.Traces))
	}
	if result.Traces[0].InputTokens != 100 {
		t.Errorf("input tokens: got %d, want 100", result.Traces[0].InputTokens)
	}
}

func TestRun_ToolTurn(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")

		if callCount == 1 {
			// First call: model wants to use a tool
			w.Write(mockAPIResponse([]ContentBlock{
				{Type: "tool_use", ID: "toolu_123", Name: "greet", Input: json.RawMessage(`{"name":"Ana"}`)},
			}, "tool_use"))
			return
		}

		// Second call: model gives final text
		w.Write(mockAPIResponse([]ContentBlock{NewTextBlock("Ana says hi!")}, "end_turn"))
	}))
	defer server.Close()

	client := testClient(server.URL)

	tools := []Tool{{
		Name:        "greet",
		Description: "Greet someone",
		InputSchema: marshalSchema(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{"type": "string"},
			},
		}),
		Execute: func(ctx context.Context, input json.RawMessage) (string, error) {
			var args struct{ Name string }
			json.Unmarshal(input, &args)
			return "Hello, " + args.Name + "!", nil
		},
	}}

	result, err := client.Run(context.Background(), RunConfig{
		Messages: []Message{NewUserMessage(NewTextBlock("Say hi to Ana"))},
		Tools:    tools,
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.Reply != "Ana says hi!" {
		t.Errorf("reply: got %q, want %q", result.Reply, "Ana says hi!")
	}
	if len(result.Traces) != 2 {
		t.Fatalf("traces: got %d, want 2", len(result.Traces))
	}
	if len(result.Traces[0].ToolCalls) != 1 {
		t.Fatalf("tool calls in trace 0: got %d, want 1", len(result.Traces[0].ToolCalls))
	}
	if result.Traces[0].ToolCalls[0].Name != "greet" {
		t.Errorf("tool call name: got %q, want %q", result.Traces[0].ToolCalls[0].Name, "greet")
	}

	// Verify tool result is in messages
	found := false
	for _, msg := range result.Messages {
		for _, block := range msg.Content {
			if block.Type == "tool_result" && block.Content == "Hello, Ana!" {
				found = true
			}
		}
	}
	if !found {
		t.Error("tool result 'Hello, Ana!' not found in messages")
	}
}

func TestRun_ToolErrorPropagated(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")

		if callCount == 1 {
			w.Write(mockAPIResponse([]ContentBlock{
				{Type: "tool_use", ID: "toolu_err", Name: "fail_tool", Input: json.RawMessage(`{}`)},
			}, "tool_use"))
			return
		}

		// Verify the tool result has is_error set
		var req apiRequest
		body := make([]byte, r.ContentLength)
		r.Body.Read(body)
		json.Unmarshal(body, &req)

		lastMsg := req.Messages[len(req.Messages)-1]
		for _, block := range lastMsg.Content {
			if block.Type == "tool_result" && !block.IsError {
				t.Error("expected is_error=true on tool result, got false")
			}
		}

		w.Write(mockAPIResponse([]ContentBlock{NewTextBlock("Tool failed, sorry.")}, "end_turn"))
	}))
	defer server.Close()

	client := testClient(server.URL)

	tools := []Tool{{
		Name:        "fail_tool",
		Description: "Always fails",
		InputSchema: marshalSchema(map[string]any{"type": "object", "properties": map[string]any{}}),
		Execute: func(ctx context.Context, input json.RawMessage) (string, error) {
			return "", fmt.Errorf("something went wrong")
		},
	}}

	result, err := client.Run(context.Background(), RunConfig{
		Messages: []Message{NewUserMessage(NewTextBlock("Do something"))},
		Tools:    tools,
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.Reply != "Tool failed, sorry." {
		t.Errorf("reply: got %q, want %q", result.Reply, "Tool failed, sorry.")
	}

	// Verify trace recorded the error
	if len(result.Traces[0].ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call in trace, got %d", len(result.Traces[0].ToolCalls))
	}
	if result.Traces[0].ToolCalls[0].Error == "" {
		t.Error("expected non-empty error in tool trace")
	}
}

func TestRun_MaxTurns(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Always request a tool call, never finish
		w.Write(mockAPIResponse([]ContentBlock{
			{Type: "tool_use", ID: "toolu_loop", Name: "noop", Input: json.RawMessage(`{}`)},
		}, "tool_use"))
	}))
	defer server.Close()

	client := testClient(server.URL)

	tools := []Tool{{
		Name:        "noop",
		Description: "Does nothing",
		InputSchema: marshalSchema(map[string]any{"type": "object", "properties": map[string]any{}}),
		Execute: func(ctx context.Context, input json.RawMessage) (string, error) {
			return "ok", nil
		},
	}}

	result, err := client.Run(context.Background(), RunConfig{
		Messages: []Message{NewUserMessage(NewTextBlock("loop"))},
		Tools:    tools,
		MaxTurns: 3,
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Traces) != 3 {
		t.Errorf("expected 3 traces (max turns), got %d", len(result.Traces))
	}
}

func TestRun_ConcurrentExecution(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req apiRequest
		body := make([]byte, r.ContentLength)
		r.Body.Read(body)
		json.Unmarshal(body, &req)

		w.Header().Set("Content-Type", "application/json")

		// First request: two tool calls
		lastMsg := req.Messages[len(req.Messages)-1]
		hasToolResult := false
		for _, block := range lastMsg.Content {
			if block.Type == "tool_result" {
				hasToolResult = true
				break
			}
		}

		if !hasToolResult {
			w.Write(mockAPIResponse([]ContentBlock{
				{Type: "tool_use", ID: "toolu_a", Name: "slow_tool", Input: json.RawMessage(`{"id":"a"}`)},
				{Type: "tool_use", ID: "toolu_b", Name: "slow_tool", Input: json.RawMessage(`{"id":"b"}`)},
			}, "tool_use"))
			return
		}

		w.Write(mockAPIResponse([]ContentBlock{NewTextBlock("Both done!")}, "end_turn"))
	}))
	defer server.Close()

	client := testClient(server.URL)

	var concurrent atomic.Int32
	var maxConcurrent atomic.Int32

	tools := []Tool{{
		Name:        "slow_tool",
		Description: "Sleeps briefly",
		InputSchema: marshalSchema(map[string]any{"type": "object", "properties": map[string]any{"id": map[string]any{"type": "string"}}}),
		Execute: func(ctx context.Context, input json.RawMessage) (string, error) {
			cur := concurrent.Add(1)
			for {
				old := maxConcurrent.Load()
				if cur <= old || maxConcurrent.CompareAndSwap(old, cur) {
					break
				}
			}
			time.Sleep(50 * time.Millisecond)
			concurrent.Add(-1)
			return "done", nil
		},
	}}

	result, err := client.Run(context.Background(), RunConfig{
		Messages: []Message{NewUserMessage(NewTextBlock("run both"))},
		Tools:    tools,
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.Reply != "Both done!" {
		t.Errorf("reply: got %q, want %q", result.Reply, "Both done!")
	}

	if maxConcurrent.Load() < 2 {
		t.Errorf("expected concurrent execution (max concurrent: %d), tools did not run in parallel", maxConcurrent.Load())
	}
}

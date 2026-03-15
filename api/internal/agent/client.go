package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

const (
	defaultBaseURL   = "https://api.anthropic.com"
	defaultModel     = "claude-sonnet-4-6"
	anthropicVersion = "2023-06-01"
)

// Client calls the Anthropic Messages API directly.
type Client struct {
	APIKey  string
	Model   string
	BaseURL string
}

// NewClient creates a Client using CLAUDE_API_KEY from env.
func NewClient() *Client {
	return &Client{
		APIKey:  os.Getenv("CLAUDE_API_KEY"),
		Model:   defaultModel,
		BaseURL: defaultBaseURL,
	}
}

// apiRequest is the request body for /v1/messages.
type apiRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	System    []apiTextBlock  `json:"system,omitempty"`
	Messages  []Message       `json:"messages"`
	Tools     []apiToolDef    `json:"tools,omitempty"`
}

type apiTextBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type apiToolDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

// apiResponse is the response body from /v1/messages.
type apiResponse struct {
	Content  []ContentBlock `json:"content"`
	StopReason string       `json:"stop_reason"`
	Usage    apiUsage       `json:"usage"`
}

type apiUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// call sends a single request to the Messages API.
func (c *Client) call(ctx context.Context, system string, messages []Message, tools []apiToolDef, maxTokens int) (*apiResponse, error) {
	req := apiRequest{
		Model:     c.Model,
		MaxTokens: maxTokens,
		Messages:  messages,
		Tools:     tools,
	}
	if system != "" {
		req.System = []apiTextBlock{{Type: "text", Text: system}}
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	baseURL := c.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-Key", c.APIKey)
	httpReq.Header.Set("Anthropic-Version", anthropicVersion)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var apiResp apiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &apiResp, nil
}

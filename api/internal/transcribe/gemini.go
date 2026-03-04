package transcribe

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const geminiEndpoint = "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent"

// Client calls the Gemini API for media transcription and description.
type Client struct {
	apiKey string
	http   *http.Client
}

// NewClient creates a Gemini transcription client.
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		http:   &http.Client{Timeout: 30 * time.Second},
	}
}

// Transcribe sends audio bytes to Gemini and returns the transcript.
// The audio should be in OGG/Opus format (WhatsApp voice notes).
func (c *Client) Transcribe(ctx context.Context, audio []byte) (string, error) {
	reqBody := map[string]any{
		"contents": []map[string]any{
			{
				"parts": []map[string]any{
					{
						"inline_data": map[string]any{
							"mime_type": "audio/ogg",
							"data":      base64.StdEncoding.EncodeToString(audio),
						},
					},
					{
						"text": "Transcreva este áudio em português. Retorne apenas a transcrição, sem comentários.",
					},
				},
			},
		},
	}
	return c.call(ctx, reqBody)
}

// DescribeImage sends image bytes to Gemini and returns a Portuguese description.
// caption is optional context provided by the sender; empty string means no caption.
func (c *Client) DescribeImage(ctx context.Context, imageBytes []byte, mimeType, caption string) (string, error) {
	prompt := "Descreva esta imagem em português de forma concisa, em uma ou duas frases."
	if caption != "" {
		prompt = fmt.Sprintf("Descreva esta imagem em português de forma concisa, em uma ou duas frases. O cliente enviou com a legenda: %q.", caption)
	}

	reqBody := map[string]any{
		"contents": []map[string]any{
			{
				"parts": []map[string]any{
					{
						"inline_data": map[string]any{
							"mime_type": mimeType,
							"data":      base64.StdEncoding.EncodeToString(imageBytes),
						},
					},
					{
						"text": prompt,
					},
				},
			},
		},
	}
	return c.call(ctx, reqBody)
}

// DescribeVideo sends video bytes to Gemini and returns a Portuguese description.
// caption is optional context provided by the sender; empty string means no caption.
func (c *Client) DescribeVideo(ctx context.Context, videoBytes []byte, mimeType, caption string) (string, error) {
	prompt := "Descreva este vídeo em português de forma concisa, em uma ou duas frases."
	if caption != "" {
		prompt = fmt.Sprintf("Descreva este vídeo em português de forma concisa, em uma ou duas frases. O cliente enviou com a legenda: %q.", caption)
	}

	reqBody := map[string]any{
		"contents": []map[string]any{
			{
				"parts": []map[string]any{
					{
						"inline_data": map[string]any{
							"mime_type": mimeType,
							"data":      base64.StdEncoding.EncodeToString(videoBytes),
						},
					},
					{
						"text": prompt,
					},
				},
			},
		},
	}
	return c.call(ctx, reqBody)
}

func (c *Client) call(ctx context.Context, reqBody any) (string, error) {
	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", geminiEndpoint+"?key="+c.apiKey, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("gemini request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("gemini API error %d: %s", resp.StatusCode, respBody)
	}

	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response from Gemini")
	}

	return result.Candidates[0].Content.Parts[0].Text, nil
}

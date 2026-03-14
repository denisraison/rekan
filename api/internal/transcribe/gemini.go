package transcribe

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png" // register PNG decoder for image.Decode
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/image/draw"
)

const geminiEndpoint = "https://generativelanguage.googleapis.com/v1beta/models/gemini-3.1-flash-lite-preview:generateContent"

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
// mimeType is the MIME type of the audio (e.g. "audio/ogg", "audio/webm", "audio/mp4").
func (c *Client) Transcribe(ctx context.Context, audio []byte, mimeType string) (string, error) {
	reqBody := map[string]any{
		"contents": []map[string]any{
			{
				"parts": []map[string]any{
					{
						"inline_data": map[string]any{
							"mime_type": mimeType,
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

const maxImageDim = 1024

// downscaleImage resizes an image so neither dimension exceeds maxImageDim.
// Returns JPEG bytes and "image/jpeg" mime type, or the original data unchanged if
// decoding fails or the image is already small enough.
func downscaleImage(data []byte, mimeType string) ([]byte, string) {
	if !strings.HasPrefix(mimeType, "image/") {
		return data, mimeType
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return data, mimeType
	}
	if cfg.Width <= maxImageDim && cfg.Height <= maxImageDim {
		return data, mimeType
	}
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return data, mimeType
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	scale := float64(maxImageDim) / float64(max(w, h))
	nw, nh := int(float64(w)*scale), int(float64(h)*scale)
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	draw.BiLinear.Scale(dst, dst.Bounds(), src, b, draw.Over, nil)
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 80}); err != nil {
		return data, mimeType
	}
	return buf.Bytes(), "image/jpeg"
}

// DescribeImage sends image bytes to Gemini and returns a Portuguese description.
// caption is optional context provided by the sender; empty string means no caption.
func (c *Client) DescribeImage(ctx context.Context, imageBytes []byte, mimeType, caption string) (string, error) {
	imageBytes, mimeType = downscaleImage(imageBytes, mimeType)

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
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body) //nolint:errcheck // best-effort read for error message
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
		return "", errors.New("empty response from Gemini")
	}

	return result.Candidates[0].Content.Parts[0].Text, nil
}

package transcribe

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func geminiResponse(text string) map[string]any {
	return map[string]any{
		"candidates": []map[string]any{
			{
				"content": map[string]any{
					"parts": []map[string]any{
						{"text": text},
					},
				},
			},
		},
	}
}

func TestDescribeImage(t *testing.T) {
	tests := []struct {
		name        string
		caption     string
		wantPrompt  string // substring expected in the request body
		serverReply string
		wantDesc    string
		serverCode  int
		wantErr     bool
	}{
		{
			name:        "returns description",
			caption:     "",
			wantPrompt:  "Descreva esta imagem",
			serverReply: "Uma foto de um prato de comida.",
			wantDesc:    "Uma foto de um prato de comida.",
			serverCode:  http.StatusOK,
		},
		{
			name:        "includes caption in prompt",
			caption:     "meu almoço",
			wantPrompt:  "meu almoço",
			serverReply: "Um prato com arroz e feijão.",
			wantDesc:    "Um prato com arroz e feijão.",
			serverCode:  http.StatusOK,
		},
		{
			name:       "returns error on non-200",
			serverCode: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedBody string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				b, err := io.ReadAll(r.Body)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				capturedBody = string(b)

				w.WriteHeader(tt.serverCode)
				if tt.serverCode == http.StatusOK {
					if err := json.NewEncoder(w).Encode(geminiResponse(tt.serverReply)); err != nil {
						return
					}
				}
			}))
			defer srv.Close()

			c := NewClient("test-key")
			// Point client at the test server by overriding the endpoint via the
			// unexported field — we patch the HTTP client's transport instead.
			c.http = &http.Client{
				Transport: rewriteHost(srv.URL),
			}

			desc, err := c.DescribeImage(context.Background(), []byte("fake-image"), "image/jpeg", tt.caption)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if desc != tt.wantDesc {
				t.Errorf("desc = %q, want %q", desc, tt.wantDesc)
			}
			if tt.wantPrompt != "" && !strings.Contains(capturedBody, tt.wantPrompt) {
				t.Errorf("request body missing %q\nbody: %s", tt.wantPrompt, capturedBody)
			}
		})
	}
}

func TestDescribeVideo(t *testing.T) {
	tests := []struct {
		name        string
		caption     string
		wantPrompt  string
		serverReply string
		wantDesc    string
		serverCode  int
		wantErr     bool
	}{
		{
			name:        "returns description",
			caption:     "",
			wantPrompt:  "Descreva este vídeo",
			serverReply: "Um vídeo de uma pessoa cozinhando.",
			wantDesc:    "Um vídeo de uma pessoa cozinhando.",
			serverCode:  http.StatusOK,
		},
		{
			name:        "includes caption in prompt",
			caption:     "novidade do dia",
			wantPrompt:  "novidade do dia",
			serverReply: "Vídeo mostrando um novo prato do restaurante.",
			wantDesc:    "Vídeo mostrando um novo prato do restaurante.",
			serverCode:  http.StatusOK,
		},
		{
			name:       "returns error on non-200",
			serverCode: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedBody string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				b, err := io.ReadAll(r.Body)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				capturedBody = string(b)

				w.WriteHeader(tt.serverCode)
				if tt.serverCode == http.StatusOK {
					if err := json.NewEncoder(w).Encode(geminiResponse(tt.serverReply)); err != nil {
						return
					}
				}
			}))
			defer srv.Close()

			c := NewClient("test-key")
			c.http = &http.Client{Transport: rewriteHost(srv.URL)}

			desc, err := c.DescribeVideo(context.Background(), []byte("fake-video"), "video/mp4", tt.caption)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if desc != tt.wantDesc {
				t.Errorf("desc = %q, want %q", desc, tt.wantDesc)
			}
			if tt.wantPrompt != "" && !strings.Contains(capturedBody, tt.wantPrompt) {
				t.Errorf("request body missing %q\nbody: %s", tt.wantPrompt, capturedBody)
			}
		})
	}
}

// rewriteHost returns a RoundTripper that redirects all requests to baseURL.
type rewriteHost string

func (base rewriteHost) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.URL.Host = strings.TrimPrefix(string(base), "http://")
	req.URL.Scheme = "http"
	return http.DefaultTransport.RoundTrip(req)
}

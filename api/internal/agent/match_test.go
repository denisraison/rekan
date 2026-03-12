package agent

import "testing"

func TestNormalizeForMatch(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"Patricia", "patricia"},
		{"Patrícia", "patricia"},
		{"  MARIA ", "maria"},
		{"São Paulo", "sao paulo"},
	}
	for _, tt := range tests {
		got := normalizeForMatch(tt.input)
		if got != tt.want {
			t.Errorf("normalizeForMatch(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

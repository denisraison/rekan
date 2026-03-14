package agent_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/denisraison/rekan/api/internal/agent"
)

func TestClassifyConfirmation(t *testing.T) {
	if os.Getenv("CLAUDE_API_KEY") == "" {
		t.Skip("CLAUDE_API_KEY not set")
	}

	cc := agent.NewClaudeClient()
	ctx := context.Background()
	action := "Cadastrar cliente Ana (Manicure, Goiânia)"

	tests := []struct {
		message string
		want    agent.ConfirmationClass
	}{
		// Confirmations
		{"beleza", agent.ClassConfirm},
		{"manda ver", agent.ClassConfirm},
		{"bora", agent.ClassConfirm},
		{"tá bom", agent.ClassConfirm},

		// Cancellations
		{"melhor não", agent.ClassCancel},
		{"deixa quieto", agent.ClassCancel},

		// Other (new instruction, not confirmation)
		{"muda a cidade pra BH", agent.ClassOther},
	}

	for _, tt := range tests {
		t.Run(tt.message, func(t *testing.T) {
			start := time.Now()
			got, err := cc.ClassifyConfirmation(ctx, tt.message, action)
			elapsed := time.Since(start)

			if err != nil {
				t.Fatalf("ClassifyConfirmation(%q) error: %v", tt.message, err)
			}
			if got != tt.want {
				t.Errorf("ClassifyConfirmation(%q) = %d, want %d", tt.message, got, tt.want)
			}
			// Haiku typically responds in ~200ms, but network overhead adds 300-500ms.
			// Use 2s as test ceiling to avoid flakes from cold connections.
			if elapsed > 2*time.Second {
				t.Errorf("ClassifyConfirmation(%q) took %v, want < 2s", tt.message, elapsed)
			}
		})
	}
}

func TestFastPath_SkipsLLM(t *testing.T) {
	// "sim" matches isConfirmation, so the LLM should never be called.
	// We verify by checking that isConfirmation returns true for "sim".
	// The actual integration is tested in TestCancellationFlow and TestCustomerCreate_HappyPath
	// which use "sim" and never set up a real Claude client.

	confirmWords := []string{"sim", "confirma", "isso", "pode fazer", "pode", "s"}
	for _, w := range confirmWords {
		if !agent.IsConfirmation(w) {
			t.Errorf("IsConfirmation(%q) = false, want true", w)
		}
	}

	cancelWords := []string{"não", "nao", "deixa", "cancela", "esquece", "n", "para"}
	for _, w := range cancelWords {
		if !agent.IsCancellation(w) {
			t.Errorf("IsCancellation(%q) = false, want true", w)
		}
	}
}

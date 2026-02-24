package postingtime

import "testing"

func TestForBusinessType(t *testing.T) {
	tests := []struct {
		input    string
		wantCat  string
		wantPrim string
	}{
		{"hamburgueria", "food", "10h-14h"},
		{"Pizzaria do Zé", "food", "10h-14h"},
		{"confeitaria artesanal", "food", "10h-14h"},
		{"salão de beleza", "beauty", "10h-13h"},
		{"Barbearia Premium", "beauty", "10h-13h"},
		{"loja de roupas femininas", "fashion", "11h-14h"},
		{"personal trainer", "fitness", "6h-8h"},
		{"studio de pilates", "fitness", "6h-8h"},
		{"petshop", "pet", "11h-13h"},
		// fallback
		{"estúdio de fotografia", "", "11h-13h"},
		{"consultoria financeira", "", "11h-13h"},
	}
	for _, tt := range tests {
		w := ForBusinessType(tt.input)
		if w.Primary != tt.wantPrim {
			t.Errorf("ForBusinessType(%q).Primary = %q, want %q", tt.input, w.Primary, tt.wantPrim)
		}
	}
}

func TestTip(t *testing.T) {
	tip := Tip("hamburgueria")
	want := "*Melhor horário pra postar:* entre 10h-14h ou entre 17h-19h"
	if tip != want {
		t.Errorf("Tip = %q, want %q", tip, want)
	}
}

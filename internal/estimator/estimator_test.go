package estimator

import (
	"testing"
)

func TestCharDiv4(t *testing.T) {
	est := CharDiv4{}

	tests := []struct {
		name       string
		input      string
		wantTokens int64
		wantLines  int64
	}{
		{
			name:       "empty",
			input:      "",
			wantTokens: 0,
			wantLines:  0,
		},
		{
			name:       "4 chars = 1 token",
			input:      "abcd",
			wantTokens: 1,
			wantLines:  1,
		},
		{
			name:       "5 chars = 2 tokens (ceil)",
			input:      "abcde",
			wantTokens: 2,
			wantLines:  1,
		},
		{
			name:       "8 chars = 2 tokens",
			input:      "abcdefgh",
			wantTokens: 2,
			wantLines:  1,
		},
		{
			name:       "multiline",
			input:      "line1\nline2\nline3",
			wantTokens: 5, // 17 chars → ceil(17/4) = 5
			wantLines:  3,
		},
		{
			name:       "trailing newline",
			input:      "hello\n",
			wantTokens: 2, // 6 chars → ceil(6/4) = 2
			wantLines:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, lines := est.Estimate([]byte(tt.input))
			if tokens != tt.wantTokens {
				t.Errorf("tokens = %d, want %d", tokens, tt.wantTokens)
			}
			if lines != tt.wantLines {
				t.Errorf("lines = %d, want %d", lines, tt.wantLines)
			}
		})
	}
}

func TestCharDiv4Deterministic(t *testing.T) {
	est := CharDiv4{}
	input := []byte("The quick brown fox jumps over the lazy dog.\nLine 2.\n")

	firstTokens, firstLines := est.Estimate(input)
	for i := 0; i < 1000; i++ {
		tokens, lines := est.Estimate(input)
		if tokens != firstTokens || lines != firstLines {
			t.Fatalf("not deterministic at iteration %d: got (%d,%d) want (%d,%d)",
				i, tokens, lines, firstTokens, firstLines)
		}
	}
}

package lexer

import (
	"io"
	"strings"
	"testing"
)

func TestPreprocesor_HandleDots_ValidForms(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "single dot", input: "."},
		{name: "ellipsis", input: "..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newPreprocesor(mockScanner(tt.input))

			tok, err := p.next()
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if tok.Type != tokenPunctuationTBD {
				t.Fatalf("expected %v, got %v", tokenPunctuationTBD, tok.Type)
			}
			if got := string(tok.Raw); got != tt.input {
				t.Fatalf("expected raw token %q, got %q", tt.input, got)
			}

			_, err = p.next()
			if err != io.EOF {
				t.Fatalf("expected EOF after token, got %v", err)
			}
		})
	}
}

func TestPreprocesor_HandleDots_InvalidForms(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "double dot", input: ".."},
		{name: "quad dot", input: "...."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newPreprocesor(mockScanner(tt.input))

			_, err := p.next()
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), "wanted '.' or '...'") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestLexer_Peek_PropagatesErrors(t *testing.T) {
	t.Run("propagates EOF", func(t *testing.T) {
		lex := newTestLexerFromString("")

		_, err := lex.Peek()
		if err != io.EOF {
			t.Fatalf("expected EOF, got %v", err)
		}
	})

	t.Run("propagates preprocessing errors", func(t *testing.T) {
		lex := newTestLexerFromString("..")

		_, err := lex.Peek()
		if err == nil {
			t.Fatal("expected preprocessing error, got nil")
		}
		if !strings.Contains(err.Error(), "wanted '.' or '...'") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func newTestLexerFromString(input string) *Lexer {
	s := mockScanner(input)
	return &Lexer{s: s, p: newPreprocesor(s)}
}

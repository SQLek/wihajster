package lexer

import (
	"io"
	"strings"
	"testing"
)

func TestReadPunctuation(t *testing.T) {
	// Rant: punctuations are nasty so they go dedicated test.
	punctuations := []string{
		"[", "]", "(", ")", "{", "}", ".", "->",
		"++", "--", "&", "*", "+", "-", "~", "!",
		"/", "%", "<<", ">>", "<", ">", "<=", ">=", "==", "!=", "^", "|", "&&", "||",
		"?", ":", ";", "...",
		"=", "*=", "/=", "%=", "+=", "-=", "<<=", ">>=", "&=", "^=", "|=",
		",", "#", "##",
		"<:", ":>", "<%", "%>", "%:", "%:%:",
	}

	for _, p := range punctuations {
		name := "punctuation " + p
		t.Run(name, func(t *testing.T) {
			firstByte := p[0]
			tokenizer := &tokenizer{
				scanner: strings.NewReader(p[1:]),
				line:    1,
				column:  0,
			}
			token, err := tokenizer.readPunctuation(firstByte)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if token.Type != TokenPunctuation {
				t.Errorf("got token type %v, want %v", token.Type, TokenPunctuation)
			}
			if token.Value != p {
				t.Errorf("got token value %q, want %q", token.Value, p)
			}
			dandling, err := tokenizer.scanner.ReadByte()
			if err != io.EOF {
				t.Errorf("expected EOF after reading punctuation, got byte %q and error %v", dandling, err)
			}

			// punctuations do quirks on collumn counting, so we check it here
			if token.Line != 1 {
				t.Errorf("got token line %d, want 1", token.Line)
			}
			if token.Column != 0 {
				t.Errorf("got token column %d, want 0", token.Column)
			}
			if tokenizer.column != len(p) {
				t.Errorf("got tokenizer column %d, want %d", tokenizer.column, len(p))
			}
		})
	}
}

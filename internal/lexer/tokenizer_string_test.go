package lexer

import (
	"strings"
	"testing"
)

func TestReadCharacterOrStringLiteral(t *testing.T) {
	tests := []struct {
		name string
		text string

		// escape sequences could throw line:column counting off
		line, column, afterLine, afterColumn int

		want string

		isString bool
	}{
		{
			name: "simple string literal",
			text: `"hello world"`,
			line: 1, column: 1, afterLine: 1, afterColumn: 14,
			want:     `"hello world"`,
			isString: true,
		},
		{
			name: "simple character literal",
			text: `'a'`,
			line: 1, column: 1, afterLine: 1, afterColumn: 4,
			want:     `'a'`,
			isString: false,
		},
		{
			// Rant: 32bit character in 4 lines, just in case you're paid by line of code
			name: "packed multiline character constant",
			text: "'a\\\nb\\\nc\\\nd'",
			line: 1, column: 1, afterLine: 4, afterColumn: 4,
			want:     "'abcd'",
			isString: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := &tokenizer{
				line:    tt.line,
				column:  tt.column,
				scanner: strings.NewReader(tt.text),
			}
			got, err := tokenizer.next()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			switch got.Type {
			case TokenStringLiteral:
				if !tt.isString {
					t.Fatalf("expected character literal, got string literal")
				}
			case TokenCharacterConstant:
				if tt.isString {
					t.Fatalf("expected string literal, got character literal")
				}
			default:
				t.Fatalf("expected string or character literal, got %v", got.Type)
			}
			if got.Value != tt.want {
				t.Errorf("unexpected token value: got %q, want %q", got.Value, tt.want)
			}
			if got.Line != tt.line {
				t.Errorf("unexpected token line: got %d, want %d", got.Line, tt.line)
			}
			if got.Column != tt.column {
				t.Errorf("unexpected token column: got %d, want %d", got.Column, tt.column)
			}
			if tokenizer.line != tt.afterLine {
				t.Errorf("unexpected tokenizer line after reading: got %d, want %d", tokenizer.line, tt.afterLine)
			}
			if tokenizer.column != tt.afterColumn {
				t.Errorf("unexpected tokenizer column after reading: got %d, want %d", tokenizer.column, tt.afterColumn)
			}
		})
	}
}

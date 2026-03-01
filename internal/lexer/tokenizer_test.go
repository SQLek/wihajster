package lexer

import (
	"strings"
	"testing"
)

// skim the surface tests for detecting obvoius problems
// and lex routing

func TestLex(t *testing.T) {
	tests := []struct {
		name  string
		input string

		expectType TokenType
		expectRaw  string
	}{
		{
			name:       "simple identifier",
			input:      "foo13",
			expectType: TokenIdentifier,
			expectRaw:  "foo13",
		},
		{
			name:       "decimal integer",
			input:      "12345",
			expectType: TokenIntegerConstant,
			expectRaw:  "12345",
		},
		{
			name:       "octal integer zero only",
			input:      "0",
			expectType: TokenIntegerConstant,
			expectRaw:  "0",
		},
		{
			name:       "octal integer with digits",
			input:      "0777",
			expectType: TokenIntegerConstant,
			expectRaw:  "0777",
		},
		{
			name:       "hexedecimal integer",
			input:      "0xDEADBEEF",
			expectType: TokenIntegerConstant,
			expectRaw:  "0xDEADBEEF",
		},
		{
			name:       "decimal integer with U suffix",
			input:      "42U",
			expectType: TokenIntegerConstant,
			expectRaw:  "42U",
		},
		{
			name:       "decimal integer with LLU suffix",
			input:      "10LLU",
			expectType: TokenIntegerConstant,
			expectRaw:  "10LLU",
		},
		{
			name:       "string literal simple",
			input:      "\"hi\"",
			expectType: TokenStringLiteral,
			expectRaw:  "\"hi\"",
		},
		{
			name:       "string literal with line continuation",
			input:      "\"hi\\\nthere\"",
			expectType: TokenStringLiteral,
			expectRaw:  "\"hithere\"",
		},
		{
			name:       "character constant simple",
			input:      "'a'",
			expectType: TokenCharacterConstant,
			expectRaw:  "'a'",
		},
		{
			name:       "two-char punctuation equality",
			input:      "==",
			expectType: tokenPunctuationTBD,
			expectRaw:  "==",
		},
		{
			name:       "single-char punctuation paren",
			input:      "(",
			expectType: tokenPunctuationTBD,
			expectRaw:  "(",
		},
		{
			name:       "preprocessor glue ##",
			input:      "##",
			expectType: tokenPreProcGlue,
			expectRaw:  "##",
		},
		{
			name:       "preprocessor start #",
			input:      "#",
			expectType: tokenPreprocStart,
			expectRaw:  "#",
		},
		{
			name:       "shift-left assign <<=",
			input:      "<<=",
			expectType: tokenPunctuationTBD,
			expectRaw:  "<<=",
		},
		{
			name:       "shift-right >>",
			input:      ">>",
			expectType: tokenShiftRight,
			expectRaw:  ">>",
		},
		{
			name:       "logical and &&",
			input:      "&&",
			expectType: tokenPunctuationTBD,
			expectRaw:  "&&",
		},
		{
			name:       "logical or ||",
			input:      "||",
			expectType: tokenPunctuationTBD,
			expectRaw:  "||",
		},
		{
			name:       "dots ellipsis variant",
			input:      "....",
			expectType: tokenDots,
			expectRaw:  "....",
		},
		{
			name:       "skip single-line comment then ident",
			input:      "// comment here\nfoo",
			expectType: TokenIdentifier,
			expectRaw:  "foo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := mockScanner(tt.input)
			var builder strings.Builder
			buildFn := mockTokenBuildFn(&builder)

			tokType, err := lex(s, buildFn)
			if err != nil {
				t.Fatal("unexpected error:", err)
			}

			tokRaw := builder.String()
			if tokType != tt.expectType || tokRaw != tt.expectRaw {
				t.Logf("want %d %q", tt.expectType, tt.expectRaw)
				t.Logf("got  %d %q", tokType, tokRaw)
				t.FailNow()
			}
		})
	}
}

func TestLex_Errors(t *testing.T) {
	tests := []struct {
		name  string
		input string

		expectErr error
	}{
		{
			name:      "decimal float not implemented",
			input:     "1.0",
			expectErr: ErrNotImplementedInV0,
		},
		{
			name:      "single-line string unknown escape not implemented",
			input:     "\"a\\t\"", // \t handled as generic escape here -> not implemented
			expectErr: ErrNotImplementedInV0,
		},
		{
			name:      "multi-line comment not implemented",
			input:     "/* x */",
			expectErr: ErrNotImplementedInV0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := mockScanner(tt.input)
			var builder strings.Builder
			buildFn := mockTokenBuildFn(&builder)

			_, err := lex(s, buildFn)
			if err == nil {
				t.Fatal("expected an error, got nil")
			}
			if err != tt.expectErr {
				t.Fatalf("expected error %v, got %v", tt.expectErr, err)
			}
		})
	}
}

func mockScanner(lines ...string) *scanner {
	data := strings.Join(lines, "\n")
	reader := strings.NewReader(data)
	return newScanner(reader, len(data))
}

func mockTokenBuildFn(builder *strings.Builder) tokenBuildFn {
	return func(b []byte) {
		builder.Write(b)
	}
}

package lexer

import (
	"io"
	"strings"
	"testing"
)

func TestTokenizer_next(t *testing.T) {
	type tCase []struct {
		name      string
		input     string
		wantValue string
		wantType  TokenType
	}

	idenCases := tCase{
		{
			name:      "simple identifier",
			input:     "hello",
			wantValue: "hello",
			wantType:  TokenIdentifier,
		},
		{
			name:      "identifier with digits",
			input:     "var123",
			wantValue: "var123",
			wantType:  TokenIdentifier,
		},
	}

	integerCases := tCase{
		{
			name:      "simple integer",
			input:     "42",
			wantValue: "42",
			wantType:  TokenIntegerConstant,
		},
		{
			name:      "octal integer",
			input:     "0755",
			wantValue: "0755",
			wantType:  TokenIntegerConstant,
		},
		{
			name:      "unsigned octal integer",
			input:     "0123u",
			wantValue: "0123u",
			wantType:  TokenIntegerConstant,
		},
		{
			name:      "hexadecimal integer",
			input:     "0x1A3F",
			wantValue: "0x1A3F",
			wantType:  TokenIntegerConstant,
		},
		{
			name:      "hexadecimal integer with 0X prefix and ll u suffixes",
			input:     "0XABCDEFllu",
			wantValue: "0XABCDEFllu",
			wantType:  TokenIntegerConstant,
		},
	}

	floatCases := tCase{
		{
			name:      "simple float",
			input:     "3.14",
			wantValue: "3.14",
			wantType:  TokenFloatingConstant,
		},
		{
			name:      "float with exponent",
			input:     "1.5e-10",
			wantValue: "1.5e-10",
			wantType:  TokenFloatingConstant,
		},
		{
			name:      "float with exponent and sign",
			input:     "2.5E+5",
			wantValue: "2.5E+5",
			wantType:  TokenFloatingConstant,
		},
		{
			name:      "decimal float with fractional and exponent",
			input:     ".14e-2",
			wantValue: ".14e-2",
			wantType:  TokenFloatingConstant,
		},
		{
			name:      "hexadecimal float with exponent",
			input:     "0x1.5p+3",
			wantValue: "0x1.5p+3",
			wantType:  TokenFloatingConstant,
		},
		{
			name:      "decimal float with exponent and suffix without fractional part",
			input:     "1e-10f",
			wantValue: "1e-10f",
			wantType:  TokenFloatingConstant,
		},
		{
			name:      "hexadecimal float with exponent and suffix without fractional part",
			input:     "0x15p+3l",
			wantValue: "0x15p+3l",
			wantType:  TokenFloatingConstant,
		},
	}

	var tests tCase
	tests = append(tests, idenCases...)
	tests = append(tests, integerCases...)
	tests = append(tests, floatCases...)

	// for dev iteration, select only one category of tests
	//tests = floatCases

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := &tokenizer{
				scanner: strings.NewReader(tt.input),
			}
			token, err := tokenizer.next()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if token.Value != tt.wantValue {
				t.Errorf("got %q, want %q", token.Value, tt.wantValue)
			}
			if token.Type != tt.wantType {
				t.Errorf("got type %v, want type %v", token.Type, tt.wantType)
			}
			b, err := tokenizer.scanner.ReadByte()
			if err != io.EOF {
				t.Errorf("expected EOF after reading token, got byte %q and error %v", b, err)
			}
		})
	}
}

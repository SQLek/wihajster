package lexer

import (
	"io"
	"os"
	"testing"
)

func TestXxx(t *testing.T) {
	fd, err := os.Open("../../examples/hello_uart.c")
	if err != nil {
		t.Fatal("error opening file: ", err)
	}
	defer fd.Close()

	s := newScanner(fd, 0)

	var accumulator []byte
	buildFn := func(data []byte) {
		accumulator = append(accumulator, data...)
	}

	var tokens []Token

	for {
		token := Token{
			Line:   s.line,
			Column: s.column,
		}
		tType, err := lex(s, buildFn)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal("tokenization failed:", err)
		}

		token.Type = tType
		token.Raw = accumulator
		tokens = append(tokens, token)
	}
}

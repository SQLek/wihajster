package lexer_test

import (
	"bufio"
	"os"
	"testing"

	"github.com/SQLek/wihajster/internal/lexer"
)

// Shotgun style tests on public API of lexer.
// Thease are not meant to test every byte of the input,
// but verify landmarks in the input are correctly tokenized,
// overal structure of the input is correct and lexer does not crash on valid input.

func TestLexer_example_hello_uart(t *testing.T) {
	fd, err := os.Open("../../examples/hello_uart.c")
	if err != nil {
		t.Fatal(err)
	}
	defer fd.Close()

	l := lexer.New(bufio.NewReader(fd))

	var tokens []lexer.Token
	for {
		token, err := l.Next()
		if err != nil {
			t.Fatal(err)
		}
		if token.Type == lexer.TokenEOF {
			break
		}
		tokens = append(tokens, token)
	}

	if len(tokens) != 28 {
		t.Fatalf("got %d tokens, want 28", len(tokens))
	}
}

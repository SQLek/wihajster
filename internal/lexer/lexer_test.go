package lexer

import (
	"io"
	"os"
	"testing"
)

// Tests in this file are meant to test public facing api.
// Thier purspose is to test landmarks of whole parsed file,
// rather than byte perfectness. They are more a shotgun.

func TestLexer_examples_hello_uart(t *testing.T) {
	fd, err := os.Open("../../examples/hello_uart.c")
	if err != nil {
		t.Fatal("error opening file: ", err)
	}
	defer fd.Close()

	lexer := NewLexer(fd)

	var tokens []Token
	for {
		tok, err := lexer.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal("tokenization failed:", err)
		}
		tokens = append(tokens, tok)
	}

	if l := len(tokens); l != 95 {
		t.Fatalf("lexer produced invalid amount of tokens")
	}
}

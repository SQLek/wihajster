package lexer_test

import (
	"io"
	"os"
	"testing"

	"github.com/SQLek/wihajster/internal/lexer"
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

	lex := lexer.NewLexer(fd)

	var tokens []lexer.Token
	for {
		tok, err := lex.Next()
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

	uartIdent := findToken(tokens, lexer.TokenIdentifier, "uart")
	if !uartIdent.IsValid() {
		t.Fatalf("ident'uart' not found in parsed dara")
	}
	if uartIdent.Line != 5 || uartIdent.Column != 20 {
		t.Fatalf("expected uart ident to be at 5:20 but it got parsed as %d:%d", uartIdent.Line, uartIdent.Column)
	}

	exclamationChar := findToken(tokens, lexer.TokenCharacterConstant, "'!'")
	if !exclamationChar.IsValid() {
		t.Fatalf("char'!' not found in parsed dara")
	}
	if exclamationChar.Line != 22 || exclamationChar.Column != 15 {
		t.Fatalf("expected uart ident to be at 22:15 but it got parsed as %d:%d", uartIdent.Line, uartIdent.Column)
	}
}

func findToken(tokens []lexer.Token, tokType lexer.TokenType, tokRaw string) lexer.Token {
	for i, tok := range tokens {
		if tok.Type != tokType {
			continue
		}
		if string(tok.Raw) != tokRaw {
			continue
		}
		return tokens[i]
	}
	return lexer.Token{}
}

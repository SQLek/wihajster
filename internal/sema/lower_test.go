package sema_test

import (
	"os"
	"strings"
	"testing"

	"github.com/SQLek/wihajster/internal/lexer"
	"github.com/SQLek/wihajster/internal/parser"
	"github.com/SQLek/wihajster/internal/sema"
	"github.com/SQLek/wihajster/internal/tac"
)

func TestLower_IfElseWhileAndExpressions(t *testing.T) {
	src := `
int main() {
	if (1 + 2 * 3 < 8 || 0) {
		while (1) {
			1 + 2;
			return 7;
		}
	} else {
		return 0;
	}
	return 1;
}
`

	mod := lowerOK(t, src)
	if len(mod.Functions) != 1 {
		t.Fatalf("expected one function, got %d", len(mod.Functions))
	}
	fn := mod.Functions[0]
	if fn.Name != "@main" {
		t.Fatalf("expected function @main, got %s", fn.Name)
	}
	if fn.ReturnType != "i32" {
		t.Fatalf("expected i32 return type, got %s", fn.ReturnType)
	}

	var out strings.Builder
	if err := tac.WriteModule(&out, mod); err != nil {
		t.Fatalf("write TAC: %v", err)
	}
	text := out.String()

	checks := []string{
		"func @main() -> i32 {",
		" = const.i32 1",
		" = mul ",
		" = add ",
		" = lt_s ",
		" = or ",
		"br ",
		"jmp .L",
		"ret ",
	}
	for _, want := range checks {
		if !strings.Contains(text, want) {
			t.Fatalf("expected TAC to contain %q, got:\n%s", want, text)
		}
	}
}

func TestLower_RejectsIdentifierWithoutDeclaration(t *testing.T) {
	src := `
int main() {
	return x;
}
`

	err := lowerErr(t, src)
	if !strings.Contains(err.Error(), "unsupported in current subset: identifiers without declarations") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLower_RequiresReturnForIntFunction(t *testing.T) {
	src := `
int main() {
	;
}
`

	err := lowerErr(t, src)
	if !strings.Contains(err.Error(), "may reach end without return") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func lowerOK(t *testing.T, src string) tac.Module {
	t.Helper()
	tu := parseOK(t, src)
	mod, err := sema.Lower(tu)
	if err != nil {
		t.Fatalf("lower failed: %v", err)
	}
	return mod
}

func lowerErr(t *testing.T, src string) error {
	t.Helper()
	tu := parseOK(t, src)
	_, err := sema.Lower(tu)
	if err == nil {
		t.Fatalf("expected lowering error but got none")
	}
	return err
}

func parseOK(t *testing.T, src string) *parser.TranslationUnit {
	t.Helper()
	lex := newLexer(t, src)
	tu, err := parser.Parse(lex)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	return tu
}

func newLexer(t *testing.T, src string) *lexer.Lexer {
	t.Helper()

	f, err := os.CreateTemp(t.TempDir(), "sema-*.c")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	if _, err := f.WriteString(src); err != nil {
		t.Fatalf("write temp source: %v", err)
	}
	if _, err := f.Seek(0, 0); err != nil {
		t.Fatalf("seek temp source: %v", err)
	}
	t.Cleanup(func() {
		_ = f.Close()
	})
	return lexer.NewLexer(f)
}

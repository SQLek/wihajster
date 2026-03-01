package parser_test

import (
	"os"
	"strings"
	"testing"

	"github.com/SQLek/wihajster/internal/lexer"
	"github.com/SQLek/wihajster/internal/parser"
)

func TestParseTranslationUnit_AcceptsMilestoneCoreSyntax(t *testing.T) {
	src := `
int main() {
	if (1 + 2 * 3 < 8 || 0) {
		return 7;
	} else {
		while (1) {
			1 + 2;
			return 0;
		}
	}
	return 1;
}
`

	tu := parseOK(t, src)
	if len(tu.Functions) != 1 {
		t.Fatalf("expected 1 function, got %d", len(tu.Functions))
	}

	fn := tu.Functions[0]
	if fn.ReturnType != parser.TypeSpecifierInt {
		t.Fatalf("expected int return type, got %v", fn.ReturnType)
	}
	if fn.Name != "main" {
		t.Fatalf("expected function name main, got %q", fn.Name)
	}
	if len(fn.Body.Statements) != 2 {
		t.Fatalf("expected 2 top-level statements in body, got %d", len(fn.Body.Statements))
	}

	ifStmt, ok := fn.Body.Statements[0].(parser.IfStatement)
	if !ok {
		t.Fatalf("expected first statement to be if, got %T", fn.Body.Statements[0])
	}
	if ifStmt.Else == nil {
		t.Fatalf("expected else branch to be present")
	}
}

func TestParseTranslationUnit_ExpressionPrecedence(t *testing.T) {
	src := `
int main() {
	return 1 + 2 * 3 == 7 || 0;
}
`

	tu := parseOK(t, src)
	ret, ok := tu.Functions[0].Body.Statements[0].(parser.ReturnStatement)
	if !ok {
		t.Fatalf("expected return statement, got %T", tu.Functions[0].Body.Statements[0])
	}

	or, ok := ret.Expression.(parser.BinaryExpression)
	if !ok || or.Op != lexer.TokenOrOr {
		t.Fatalf("expected top-level logical-or expression, got %#v", ret.Expression)
	}
	eq, ok := or.LHS.(parser.BinaryExpression)
	if !ok || eq.Op != lexer.TokenEq {
		t.Fatalf("expected lhs to be equality expression, got %#v", or.LHS)
	}
	add, ok := eq.LHS.(parser.BinaryExpression)
	if !ok || add.Op != lexer.TokenPlus {
		t.Fatalf("expected equality lhs to be addition expression, got %#v", eq.LHS)
	}
	mul, ok := add.RHS.(parser.BinaryExpression)
	if !ok || mul.Op != lexer.TokenStar {
		t.Fatalf("expected addition rhs to be multiplication expression, got %#v", add.RHS)
	}
}

func TestParseTranslationUnit_RejectsUnsupportedSyntax(t *testing.T) {
	tests := []struct {
		name string
		src  string
		msg  string
	}{
		{
			name: "pointer declarator",
			src: `
int *main() {
	return 0;
}
`,
			msg: "unsupported in current subset: pointers",
		},
		{
			name: "struct declaration",
			src: `
struct S main() {
	return 0;
}
`,
			msg: "unsupported in current subset: struct declarations",
		},
		{
			name: "local declaration",
			src: `
int main() {
	int x;
	return 0;
}
`,
			msg: "unsupported in current subset: declarations beyond current subset",
		},
		{
			name: "function parameters",
			src: `
int main(int x) {
	return x;
}
`,
			msg: "unsupported in current subset: function parameters",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := parseErr(t, tc.src)
			if !strings.Contains(err.Error(), tc.msg) {
				t.Fatalf("expected error to contain %q, got %q", tc.msg, err.Error())
			}
			if err.Line <= 0 || err.Column <= 0 {
				t.Fatalf("expected parser error position to be populated, got %d:%d", err.Line, err.Column)
			}
		})
	}
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

func parseErr(t *testing.T, src string) *parser.Error {
	t.Helper()
	lex := newLexer(t, src)
	_, err := parser.Parse(lex)
	if err == nil {
		t.Fatalf("expected parse error but got none")
	}
	pErr, ok := err.(*parser.Error)
	if !ok {
		t.Fatalf("expected parser.Error, got %T (%v)", err, err)
	}
	return pErr
}

func newLexer(t *testing.T, src string) *lexer.Lexer {
	t.Helper()

	f, err := os.CreateTemp(t.TempDir(), "parser-*.c")
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

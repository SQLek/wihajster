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
int add(int a, int b) {
	int tmp = a + b;
	return tmp;
}

int main() {
	int i = 0;
	int acc = 0;
	for (i = 0; i < 4; i = i + 1) {
		acc = add(acc, i);
	}
	if (acc > 0) {
		return acc;
	} else {
		while (acc < 100) {
			acc = acc + 1;
			return acc;
		}
	}
	return 0;
}
`

	tu := parseOK(t, src)
	if len(tu.Functions) != 2 {
		t.Fatalf("expected 2 functions, got %d", len(tu.Functions))
	}
	if len(tu.Declarations) != 0 {
		t.Fatalf("expected no global declarations, got %d", len(tu.Declarations))
	}

	mainFn := tu.Functions[1]
	if mainFn.ReturnType.Specifier != parser.TypeSpecifierInt {
		t.Fatalf("expected int return type, got %v", mainFn.ReturnType.Specifier)
	}
	if len(mainFn.Body.Statements) < 3 {
		t.Fatalf("expected statements in main body, got %d", len(mainFn.Body.Statements))
	}
}

func TestParseTranslationUnit_ParsesGlobalDeclarationsAndParams(t *testing.T) {
	src := `
char g = 'a';
int *gp;

int main(int argc, char *argv) {
	return argc;
}
`

	tu := parseOK(t, src)
	if len(tu.Declarations) != 2 {
		t.Fatalf("expected 2 globals, got %d", len(tu.Declarations))
	}
	if tu.Declarations[0].Type.Specifier != parser.TypeSpecifierChar {
		t.Fatalf("expected first global to be char, got %v", tu.Declarations[0].Type.Specifier)
	}
	if tu.Declarations[1].Type.PointerDepth != 1 {
		t.Fatalf("expected pointer depth 1 for gp, got %d", tu.Declarations[1].Type.PointerDepth)
	}
	if len(tu.Functions) != 1 {
		t.Fatalf("expected 1 function, got %d", len(tu.Functions))
	}
	if len(tu.Functions[0].Parameters) != 2 {
		t.Fatalf("expected 2 params, got %d", len(tu.Functions[0].Parameters))
	}
}

func TestParseTranslationUnit_ExpressionPrecedenceAndCalls(t *testing.T) {
	src := `
int main() {
	return foo(1 + 2 * 3, 7) == 0 || 1;
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
	call, ok := eq.LHS.(parser.CallExpression)
	if !ok {
		t.Fatalf("expected lhs to contain call expression, got %#v", eq.LHS)
	}
	if len(call.Args) != 2 {
		t.Fatalf("expected 2 call args, got %d", len(call.Args))
	}
}

func TestParseTranslationUnit_ParsesForStatement(t *testing.T) {
	src := `
int main() {
	for (int i = 0; i < 3; i = i + 1) {
		;
	}
	return 0;
}
`

	tu := parseOK(t, src)
	if len(tu.Functions) != 1 {
		t.Fatalf("expected 1 function, got %d", len(tu.Functions))
	}
	if _, ok := tu.Functions[0].Body.Statements[0].(parser.ForStatement); !ok {
		t.Fatalf("expected first statement to be for, got %T", tu.Functions[0].Body.Statements[0])
	}
}

func TestParseTranslationUnit_RejectsUnsupportedSyntax(t *testing.T) {
	tests := []struct {
		name string
		src  string
		msg  string
	}{
		{
			name: "variadic function",
			src: `
int main(int x, ...) {
	return x;
}
`,
			msg: "unsupported in current subset: variadic functions",
		},
		{
			name: "array declaration",
			src: `
int arr[4];
int main() { return 0; }
`,
			msg: "unsupported in current subset: arrays",
		},
		{
			name: "switch statement",
			src: `
int main() {
	switch (1) { return 0; }
}
`,
			msg: "unsupported in current subset: switch statements",
		},
		{
			name: "cast expression",
			src: `
int main() {
	return (int)1;
}
`,
			msg: "unsupported in current subset: casts",
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

func TestParserBacklogSkeleton_GroupsPresent(t *testing.T) {
	t.Run("declarators", func(t *testing.T) {})
	t.Run("declarations", func(t *testing.T) {})
	t.Run("assignment", func(t *testing.T) {})
	t.Run("calls", func(t *testing.T) {})
	t.Run("for", func(t *testing.T) {})
	t.Run("unsupported-diagnostics", func(t *testing.T) {})
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
	pErrs := parseErrors(t, src)
	if len(pErrs.Diagnostics) == 0 {
		t.Fatalf("expected parser diagnostics but got none (fatal lexer=%v)", pErrs.FatalLexer)
	}
	return pErrs.Diagnostics[0]
}

func parseErrors(t *testing.T, src string) *parser.ParseErrors {
	t.Helper()
	lex := newLexer(t, src)
	_, err := parser.Parse(lex)
	if err == nil {
		t.Fatalf("expected parse error but got none")
	}
	pErrs, ok := err.(*parser.ParseErrors)
	if !ok {
		t.Fatalf("expected *parser.ParseErrors, got %T (%v)", err, err)
	}
	return pErrs
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

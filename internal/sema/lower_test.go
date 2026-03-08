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

func TestLower_LocalsAssignmentsAndMemoryOps(t *testing.T) {
	src := `
int main() {
	int x = 1;
	x = x + 2;
	return x;
}
`

	text := lowerText(t, src)
	checks := []string{
		"func @main() -> i32 {",
		" = alloca i32",
		"store ",
		" = load ",
		" = add ",
		"ret ",
	}
	for _, want := range checks {
		if !strings.Contains(text, want) {
			t.Fatalf("expected TAC to contain %q, got:\n%s", want, text)
		}
	}
}

func TestLower_ParamReassignmentUsesSlot(t *testing.T) {
	src := `
int main(int a) {
	a = a + 1;
	return a;
}
`

	text := lowerText(t, src)
	checks := []string{
		"func @main(%a:i32) -> i32 {",
		" = alloca i32",
		"store ",
		" = load ",
	}
	for _, want := range checks {
		if !strings.Contains(text, want) {
			t.Fatalf("expected TAC to contain %q, got:\n%s", want, text)
		}
	}
}

func TestLower_ForInitDeclarationScope(t *testing.T) {
	src := `
int main() {
	for (int i = 0; i < 3; i = i + 1) {
	}
	return i;
}
`

	err := lowerErr(t, src)
	if !strings.Contains(err.Error(), "use of undeclared identifier i") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLower_AllowsShadowingInNestedBlock(t *testing.T) {
	src := `
int main() {
	int x = 1;
	{
		int x = 2;
		x = x + 1;
	}
	return x;
}
`

	_ = lowerOK(t, src)
}

func TestLower_RejectsSameScopeRedeclaration(t *testing.T) {
	src := `
int main() {
	int x = 1;
	int x = 2;
	return x;
}
`

	err := lowerErr(t, src)
	if !strings.Contains(err.Error(), "redeclared in this scope") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLower_AcceptsPrototypeAndMatchingDefinition(t *testing.T) {
	src := `
int id(int x);
int id(int x) {
	return x;
}
int main() {
	return 0;
}
`

	mod := lowerOK(t, src)
	if len(mod.Functions) != 2 {
		t.Fatalf("expected 2 lowered functions, got %d", len(mod.Functions))
	}
}

func TestLower_RejectsConflictingPrototypeDefinition(t *testing.T) {
	src := `
int id(int *x);
int id(int x) {
	return 0;
}
`

	err := lowerErr(t, src)
	if !strings.Contains(err.Error(), "does not match prototype") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLower_RejectsGlobalDeclarationsInM1(t *testing.T) {
	src := `
int g = 1;
int main() {
	return 0;
}
`

	err := lowerErr(t, src)
	if !strings.Contains(err.Error(), "unsupported in current subset: global declarations") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLower_LowersDirectFunctionCall(t *testing.T) {
	src := `
int inc(int x) {
	return x + 1;
}
int main() {
	return inc(41);
}
`

	text := lowerText(t, src)
	if !strings.Contains(text, "call @inc(") {
		t.Fatalf("expected call lowering, got:\n%s", text)
	}
}

func TestLower_RejectsCallToUndeclaredFunction(t *testing.T) {
	src := `
int main() {
	return foo(1);
}
`

	err := lowerErr(t, src)
	if !strings.Contains(err.Error(), "call to undeclared function foo") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLower_RejectsWrongCallArity(t *testing.T) {
	src := `
int f(int x) {
	return x;
}
int main() {
	return f(1, 2);
}
`

	err := lowerErr(t, src)
	if !strings.Contains(err.Error(), "expects 1 arguments, got 2") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLower_RejectsCallArgumentTypeMismatch(t *testing.T) {
	src := `
int f(int *x) {
	return 0;
}
int main() {
	int v = 0;
	return f(v);
}
`

	err := lowerErr(t, src)
	if !strings.Contains(err.Error(), "argument 1 to f has type i32, expected ptr") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLower_RejectsNonIdentifierCallTarget(t *testing.T) {
	src := `
int f(int x) {
	return x;
}
int main() {
	return (f + 0)(1);
}
`

	err := lowerErr(t, src)
	if !strings.Contains(err.Error(), "unsupported in current subset: function call target") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLower_VoidCallAllowedInExpressionStatement(t *testing.T) {
	src := `
void ping() {
	return;
}
int main() {
	ping();
	return 0;
}
`

	_ = lowerOK(t, src)
}

func TestLower_RejectsVoidCallInValueContext(t *testing.T) {
	src := `
void ping() {
	return;
}
int main() {
	return ping();
}
`

	err := lowerErr(t, src)
	if !strings.Contains(err.Error(), "return type mismatch") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLower_CharacterLiteralSimpleAndEscaped(t *testing.T) {
	src := `
int main() {
	int a = 'a';
	int b = '\n';
	int c = '\'';
	return a + b + c;
}
`

	text := lowerText(t, src)
	for _, want := range []string{"const.i32 97", "const.i32 10", "const.i32 39"} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected TAC to contain %q, got:\n%s", want, text)
		}
	}
}

func TestLower_RejectsMalformedCharacterLiteral(t *testing.T) {
	src := `
int main() {
	int a = 'ab';
	return a;
}
`

	err := lowerErr(t, src)
	if !strings.Contains(err.Error(), "multi-character literals are unsupported") {
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

func lowerText(t *testing.T, src string) string {
	t.Helper()
	mod := lowerOK(t, src)
	var out strings.Builder
	if err := tac.WriteModule(&out, mod); err != nil {
		t.Fatalf("write TAC: %v", err)
	}
	return out.String()
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

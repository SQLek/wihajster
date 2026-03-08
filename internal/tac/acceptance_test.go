package tac_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SQLek/wihajster/internal/lexer"
	"github.com/SQLek/wihajster/internal/parser"
	"github.com/SQLek/wihajster/internal/sema"
	"github.com/SQLek/wihajster/internal/tac"
)

func TestCompileAndEvaluate_Fibonacci(t *testing.T) {
	srcPath := filepath.Join("..", "..", "examples", "fibonacci.c")
	f, err := os.Open(srcPath)
	if err != nil {
		t.Fatalf("open %s: %v", srcPath, err)
	}
	defer f.Close()

	tu, err := parser.Parse(lexer.NewLexer(f))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	mod, err := sema.Lower(tu)
	if err != nil {
		t.Fatalf("lower: %v", err)
	}

	cases := []struct {
		n    int32
		want int32
	}{
		{n: 0, want: 0},
		{n: 1, want: 1},
		{n: 2, want: 1},
		{n: 5, want: 5},
		{n: 10, want: 55},
	}

	for _, tc := range cases {
		got, err := tac.EvaluateFunction(mod, "@fib", []int32{tc.n}, tac.EvalOptions{})
		if err != nil {
			t.Fatalf("evaluate fib(%d): %v", tc.n, err)
		}
		if got != tc.want {
			t.Fatalf("fib(%d): expected %d, got %d", tc.n, tc.want, got)
		}
	}

	mainRet, err := tac.EvaluateFunction(mod, "@main", nil, tac.EvalOptions{})
	if err != nil {
		t.Fatalf("evaluate @main: %v", err)
	}
	if mainRet != 55 {
		t.Fatalf("expected @main return 55, got %d", mainRet)
	}
}

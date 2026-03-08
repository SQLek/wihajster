package backend

import (
	"strings"
	"testing"

	"github.com/SQLek/wihajster/internal/tac"
)

func TestBuildFunctionViewUsesCFGBlocks(t *testing.T) {
	src := `.tac v1
func @main() -> i32 {
  %c = const.i32 1
  br %c, .Lthen, .Lelse
.Lthen:
  ret 1
.Lelse:
  ret 2
}
`
	mod, err := tac.ParseModule(strings.NewReader(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	view, err := BuildFunctionView(mod.Functions[0])
	if err != nil {
		t.Fatalf("build function view: %v", err)
	}
	if len(view.Blocks) != 3 {
		t.Fatalf("expected 3 cfg blocks, got %d", len(view.Blocks))
	}
}

func TestBuildFunctionView_ValidatesIR(t *testing.T) {
	fn := tac.Function{Name: "@bad", ReturnType: "i32", Instructions: []tac.Instruction{
		{Kind: tac.InstructionLabel, Label: ".L0"},
		{Kind: tac.InstructionJmp, TrueLabel: tac.Label(".Lmissing")},
	}}

	_, err := BuildFunctionView(fn)
	if err == nil || !strings.Contains(err.Error(), "undefined label") {
		t.Fatalf("expected undefined label error, got %v", err)
	}
}

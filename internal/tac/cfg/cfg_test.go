package cfg

import (
	"reflect"
	"strings"
	"testing"

	"github.com/SQLek/wihajster/internal/tac"
)

func TestBuildCFGBlocksAndEdges(t *testing.T) {
	src := `.tac v1
func @main() -> i32 {
  %c = const.i32 1
  br %c, .Ltrue, .Lfalse
  %x = const.i32 99
.Ltrue:
  %a = const.i32 7
  jmp .Lexit
.Lfalse:
  %b = const.i32 8
.Lexit:
  ret %a
}
`

	mod, err := tac.ParseModule(strings.NewReader(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	g, err := Build(mod.Functions[0])
	if err != nil {
		t.Fatalf("build cfg: %v", err)
	}
	if len(g.Blocks) != 5 {
		t.Fatalf("expected 5 blocks, got %d", len(g.Blocks))
	}

	if got, want := g.Blocks[0].Successors, []int{2, 3}; !reflect.DeepEqual(got, want) {
		t.Fatalf("block 0 successors mismatch: got %v want %v", got, want)
	}
	if len(g.Blocks[1].Predecessors) != 0 {
		t.Fatalf("block 1 predecessors mismatch: got %v want []", g.Blocks[1].Predecessors)
	}
	if got, want := g.Blocks[2].Successors, []int{4}; !reflect.DeepEqual(got, want) {
		t.Fatalf("block 2 successors mismatch: got %v want %v", got, want)
	}
	if got, want := g.Blocks[3].Successors, []int{4}; !reflect.DeepEqual(got, want) {
		t.Fatalf("block 3 successors mismatch: got %v want %v", got, want)
	}
	if got, want := g.Blocks[4].Predecessors, []int{2, 3}; !reflect.DeepEqual(got, want) {
		t.Fatalf("block 4 predecessors mismatch: got %v want %v", got, want)
	}
}

func TestBuildCFGRejectsUndefinedTarget(t *testing.T) {
	fn := tac.Function{
		Name: "@f",
		Instructions: []tac.Instruction{
			{Kind: tac.InstructionJmp, TrueLabel: tac.Label(".Lmissing")},
			{Kind: tac.InstructionRet},
		},
	}

	_, err := Build(fn)
	if err == nil || !strings.Contains(err.Error(), "undefined label") {
		t.Fatalf("expected undefined label error, got %v", err)
	}
}

func TestBuildCFGRejectsMidBlockTerminator(t *testing.T) {
	fn := tac.Function{
		Name: "@f",
		Instructions: []tac.Instruction{
			{Kind: tac.InstructionRet},
			{Kind: tac.InstructionOp, Opcode: tac.OpcodeConstI32, HasDestination: true, Destination: tac.Temp("%v"), Operands: []tac.Operand{tac.Immediate("1")}},
		},
	}

	_, err := Build(fn)
	if err == nil || !strings.Contains(err.Error(), "has no terminator and no fallthrough successor") {
		t.Fatalf("expected terminator placement error, got %v", err)
	}
}

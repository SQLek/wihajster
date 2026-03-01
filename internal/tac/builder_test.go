package tac

import (
	"strings"
	"testing"
)

func TestFunction_NewTemp_Sequential(t *testing.T) {
	fn := &Function{}

	if got := fn.NewTemp(); got != "%t0" {
		t.Fatalf("expected first temp %%t0, got %s", got)
	}
	if got := fn.NewTemp(); got != "%t1" {
		t.Fatalf("expected second temp %%t1, got %s", got)
	}
	if got := fn.NewTemp(); got != "%t2" {
		t.Fatalf("expected third temp %%t2, got %s", got)
	}
}

func TestFunction_AddInstruction_AppendsValueProducingOp(t *testing.T) {
	fn := &Function{}

	dst := fn.AddInstruction("add", "%a", "%b")
	if dst != "%t0" {
		t.Fatalf("expected destination %%t0, got %s", dst)
	}

	if len(fn.Instructions) != 1 {
		t.Fatalf("expected 1 instruction, got %d", len(fn.Instructions))
	}

	inst := fn.Instructions[0]
	if inst.Kind != InstructionOp {
		t.Fatalf("expected InstructionOp, got %d", inst.Kind)
	}
	if inst.Destination != "%t0" {
		t.Fatalf("expected destination %%t0, got %s", inst.Destination)
	}
	if inst.Opcode != "add" {
		t.Fatalf("expected opcode add, got %s", inst.Opcode)
	}
	if len(inst.Operands) != 2 || inst.Operands[0] != "%a" || inst.Operands[1] != "%b" {
		t.Fatalf("unexpected operands: %#v", inst.Operands)
	}

	next := fn.AddInstruction("sub", dst, "1")
	if next != "%t1" {
		t.Fatalf("expected second destination %%t1, got %s", next)
	}
}

func TestFunction_Builder_WriteParseRoundTrip(t *testing.T) {
	fn := Function{Name: "@main", ReturnType: "i32"}
	fn.AddLabel(".L0")
	v := fn.AddInstruction("const.i32", "42")
	fn.AddRet(v)

	mod := Module{Functions: []Function{fn}}
	var out strings.Builder
	if err := WriteModule(&out, mod); err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}

	reparsed, err := ParseModule(strings.NewReader(out.String()))
	if err != nil {
		t.Fatalf("unexpected reparse error: %v", err)
	}

	if len(reparsed.Functions) != 1 {
		t.Fatalf("expected 1 function after round-trip, got %d", len(reparsed.Functions))
	}

	got := reparsed.Functions[0].Instructions
	if len(got) != 3 {
		t.Fatalf("expected 3 instructions after round-trip, got %d", len(got))
	}
	if got[0].Kind != InstructionLabel || got[0].Label != ".L0" {
		t.Fatalf("unexpected label instruction: %#v", got[0])
	}
	if got[1].Kind != InstructionOp || got[1].Destination != "%t0" || got[1].Opcode != "const.i32" {
		t.Fatalf("unexpected op instruction: %#v", got[1])
	}
	if len(got[1].Operands) != 1 || got[1].Operands[0] != "42" {
		t.Fatalf("unexpected op operands: %#v", got[1].Operands)
	}
	if got[2].Kind != InstructionRet || got[2].ReturnValue != "%t0" {
		t.Fatalf("unexpected ret instruction: %#v", got[2])
	}
}

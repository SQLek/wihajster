package tac

import (
	"strings"
	"testing"
)

func TestFunction_NewTemp_Sequential(t *testing.T) {
	fn := &Function{}

	if got := fn.NewTemp(); got.Text != "%t0" {
		t.Fatalf("expected first temp %%t0, got %s", got.Text)
	}
	if got := fn.NewTemp(); got.Text != "%t1" {
		t.Fatalf("expected second temp %%t1, got %s", got.Text)
	}
	if got := fn.NewTemp(); got.Text != "%t2" {
		t.Fatalf("expected third temp %%t2, got %s", got.Text)
	}
}

func TestFunction_AddInstruction_AppendsValueProducingOp(t *testing.T) {
	fn := &Function{}

	dst := fn.AddInstruction(OpcodeAdd, Param("%a"), Param("%b"))
	if dst.Text != "%t0" {
		t.Fatalf("expected destination %%t0, got %s", dst.Text)
	}

	inst := fn.Instructions[0]
	if inst.Kind != InstructionOp || inst.Opcode != OpcodeAdd {
		t.Fatalf("unexpected instruction: %#v", inst)
	}
	if !inst.HasDestination || inst.Destination.Text != "%t0" {
		t.Fatalf("unexpected destination: %#v", inst.Destination)
	}
	if len(inst.Operands) != 2 || inst.Operands[0].Text != "%a" || inst.Operands[1].Text != "%b" {
		t.Fatalf("unexpected operands: %#v", inst.Operands)
	}
}

func TestFunction_AddCall_FormatsOperandDeterministically(t *testing.T) {
	fn := &Function{}
	dst := fn.AddCall(FunctionSymbol("@sum"), Param("%a"), Param("%b"))
	if dst.Text != "%t0" {
		t.Fatalf("expected destination %%t0, got %s", dst.Text)
	}
	inst := fn.Instructions[0]
	if inst.Opcode != OpcodeCall {
		t.Fatalf("expected call opcode, got %s", inst.Opcode)
	}
	if len(inst.Operands) != 3 || inst.Operands[0].Text != "@sum" {
		t.Fatalf("unexpected call operands: %#v", inst.Operands)
	}
}

func TestFunction_Builder_WriteParseRoundTrip(t *testing.T) {
	fn := Function{Name: "@main", ReturnType: "i32"}
	fn.AddLabel(".L0")
	v := fn.AddInstruction(OpcodeConstI32, Immediate("42"))
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
	got := reparsed.Functions[0].Instructions
	if got[1].Opcode != OpcodeConstI32 || got[1].Operands[0].Text != "42" {
		t.Fatalf("unexpected op instruction: %#v", got[1])
	}
	if !got[2].HasReturnValue || got[2].ReturnValue.Text != "%t0" {
		t.Fatalf("unexpected ret instruction: %#v", got[2])
	}
}

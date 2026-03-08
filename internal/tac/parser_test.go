package tac

import (
	"strings"
	"testing"
)

func TestParseModule_BasicExample(t *testing.T) {
	input := `.tac v1

func @add(%a:i32, %b:i32) -> i32 {
.L0:
  %t0 = add %a, %b
  ret %t0
}
`

	mod, err := ParseModule(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	if len(mod.Functions) != 1 {
		t.Fatalf("expected 1 function, got %d", len(mod.Functions))
	}
	fn := mod.Functions[0]
	if fn.Name != "@add" {
		t.Fatalf("expected @add, got %s", fn.Name)
	}
	if len(fn.Instructions) != 3 {
		t.Fatalf("expected 3 instructions, got %d", len(fn.Instructions))
	}
}

func TestParseModule_DestinationRedefinition(t *testing.T) {
	input := `.tac v1

func @bad() -> i32 {
.L0:
  %t0 = const.i32 1
  %t0 = add %t0, 2
  ret %t0
}
`

	_, err := ParseModule(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected redefinition error")
	}
	if !strings.Contains(err.Error(), "redefined") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseModule_OptionalOpcodeRejectedInM1(t *testing.T) {
	input := `.tac v1

func @mem_demo() -> i32 {
.L0:
  %t0 = phi %a, %b
  ret %t0
}
`

	_, err := ParseModule(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected optional opcode rejection")
	}
	if !strings.Contains(err.Error(), "not enabled") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWriteModule_RoundTrip(t *testing.T) {
	original := `.tac v1

func @main() -> i32 {
.L0:
  %t0 = const.i32 42
  ret %t0
}
`

	mod, err := ParseModule(strings.NewReader(original))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	var out strings.Builder
	if err := WriteModule(&out, mod); err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}

	reparsed, err := ParseModule(strings.NewReader(out.String()))
	if err != nil {
		t.Fatalf("unexpected reparse error: %v", err)
	}
	if len(reparsed.Functions) != 1 || reparsed.Functions[0].Name != "@main" {
		t.Fatalf("unexpected round-trip module: %+v", reparsed)
	}
}

func TestParseModule_CallOperands_ZeroOneManyArgs(t *testing.T) {
	input := `.tac v1

func @zero() -> i32 {
.L0:
  %t0 = call @callee0()
  ret %t0
}

func @one(%a:i32) -> i32 {
.L0:
  %t0 = call @callee1(%a)
  ret %t0
}

func @many(%a:i32, %b:i32) -> i32 {
.L0:
  %t0 = call @calleeN(%a, 1, %b)
  ret %t0
}
`

	mod, err := ParseModule(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	zero := mod.Functions[0].Instructions[1]
	if zero.CallCallee != "@callee0" || len(zero.CallArgs) != 0 {
		t.Fatalf("unexpected zero-arg call: %#v", zero)
	}

	one := mod.Functions[1].Instructions[1]
	if one.CallCallee != "@callee1" || len(one.CallArgs) != 1 || one.CallArgs[0].Text != "%a" {
		t.Fatalf("unexpected one-arg call: %#v", one)
	}

	many := mod.Functions[2].Instructions[1]
	if many.CallCallee != "@calleeN" || len(many.CallArgs) != 3 {
		t.Fatalf("unexpected many-arg call: %#v", many)
	}
	if many.CallArgs[0].Text != "%a" || many.CallArgs[1].Text != "1" || many.CallArgs[2].Text != "%b" {
		t.Fatalf("unexpected many-arg call args: %#v", many.CallArgs)
	}
}

func TestParseModule_CallMalformedTextualSyntax(t *testing.T) {
	tests := []struct {
		name string
		call string
	}{
		{name: "missing open", call: "call @f"},
		{name: "missing close", call: "call @f(%a"},
		{name: "missing callee", call: "call (%a)"},
		{name: "trailing tokens", call: "call @f() junk"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			input := `.tac v1

func @bad(%a:i32) -> i32 {
.L0:
  %t0 = ` + tc.call + `
  ret %t0
}
`
			_, err := ParseModule(strings.NewReader(input))
			if err == nil || !strings.Contains(err.Error(), "malformed call") {
				t.Fatalf("expected malformed call error, got %v", err)
			}
		})
	}
}

func TestWriteModule_CallRoundTripStructuredFields(t *testing.T) {
	mod := Module{Functions: []Function{{
		Name:       "@main",
		ReturnType: "i32",
		Instructions: []Instruction{
			{Kind: InstructionLabel, Label: ".L0"},
			{Kind: InstructionOp, HasDestination: true, Destination: Temp("%t0"), Opcode: OpcodeCall, CallCallee: "@f", CallArgs: []ValueRef{Immediate("1"), Temp("%t2")}},
			{Kind: InstructionRet, HasReturnValue: true, ReturnValue: Temp("%t0")},
		},
	}}}

	var out strings.Builder
	if err := WriteModule(&out, mod); err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}
	if !strings.Contains(out.String(), "%t0 = call @f(1, %t2)") {
		t.Fatalf("expected textual call syntax, got:\n%s", out.String())
	}

	reparsed, err := ParseModule(strings.NewReader(out.String()))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	inst := reparsed.Functions[0].Instructions[1]
	if inst.CallCallee != "@f" || len(inst.CallArgs) != 2 {
		t.Fatalf("unexpected reparsed call: %#v", inst)
	}
}

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

func TestParseModule_CallSyntax_ZeroOneManyArgs(t *testing.T) {
	input := `.tac v1

func @main(%a:i32, %b:i32) -> i32 {
.L0:
  %t0 = call @zero()
  %t1 = call @one(%a)
  %t2 = call @many(%a, %b, %t1)
  ret %t2
}
`

	mod, err := ParseModule(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	insts := mod.Functions[0].Instructions
	if insts[1].CallCallee != "@zero" || len(insts[1].CallArgs) != 0 {
		t.Fatalf("unexpected zero-arg call: %+v", insts[1])
	}
	if insts[2].CallCallee != "@one" || len(insts[2].CallArgs) != 1 || insts[2].CallArgs[0] != ValueRef("%a") {
		t.Fatalf("unexpected one-arg call: %+v", insts[2])
	}
	if insts[3].CallCallee != "@many" || len(insts[3].CallArgs) != 3 {
		t.Fatalf("unexpected many-arg call: %+v", insts[3])
	}
}

func TestParseModule_MalformedCallSyntax(t *testing.T) {
	tests := []string{
		"%t0 = call @f",
		"%t0 = call @f(",
		"%t0 = call f()",
		"%t0 = call @f(%a,, %b)",
		"%t0 = call @f())",
	}
	for _, callLine := range tests {
		t.Run(callLine, func(t *testing.T) {
			input := ".tac v1\n\nfunc @main(%a:i32, %b:i32) -> i32 {\n.L0:\n  " + callLine + "\n  ret 0\n}\n"
			_, err := ParseModule(strings.NewReader(input))
			if err == nil || !strings.Contains(err.Error(), "malformed call") {
				t.Fatalf("expected malformed call error, got %v", err)
			}
		})
	}
}

func TestWriteModule_PreservesCallTextSyntaxFromStructuredFields(t *testing.T) {
	fn := Function{Name: "@main", ReturnType: "i32"}
	fn.AddLabel(".L0")
	fn.AddCallVoid("@zero")
	v := fn.AddCall("@sum", "%x", "3")
	fn.AddRet(v)
	mod := Module{Functions: []Function{fn}}

	var out strings.Builder
	if err := WriteModule(&out, mod); err != nil {
		t.Fatalf("write: %v", err)
	}
	text := out.String()
	if !strings.Contains(text, "call @zero()") {
		t.Fatalf("missing zero-arg call text: %s", text)
	}
	if !strings.Contains(text, "call @sum(%x, 3)") {
		t.Fatalf("missing n-arg call text: %s", text)
	}
}

func TestWriteModule_CallLegacyOperandFallback(t *testing.T) {
	mod := Module{Functions: []Function{{
		Name:       "@main",
		ReturnType: "i32",
		Instructions: []Instruction{
			{Kind: InstructionLabel, Label: ".L0"},
			{Kind: InstructionOp, Opcode: "call", Operands: []string{"@ping()"}},
			{Kind: InstructionRet, ReturnValue: "0"},
		},
	}}}

	var out strings.Builder
	if err := WriteModule(&out, mod); err != nil {
		t.Fatalf("write: %v", err)
	}
	if !strings.Contains(out.String(), "call @ping()") {
		t.Fatalf("missing fallback call text: %s", out.String())
	}
}

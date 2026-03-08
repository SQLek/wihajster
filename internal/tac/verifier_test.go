package tac

import (
	"strings"
	"testing"
)

func TestVerifyInstruction_InvalidOperandKinds(t *testing.T) {
	tests := []struct {
		name string
		inst Instruction
		sub  string
	}{
		{"load with immediate", Instruction{Kind: InstructionOp, Opcode: OpcodeLoad, HasDestination: true, Destination: Temp("%t0"), Operands: []Operand{Immediate("1")}}, "stack slot pointer"},
		{"store with bad pointer", Instruction{Kind: InstructionOp, Opcode: OpcodeStore, Operands: []Operand{Temp("%t0"), Immediate("1")}}, "stack slot pointer"},
		{"call with non symbol callee", Instruction{Kind: InstructionOp, Opcode: OpcodeCall, HasDestination: true, Destination: Temp("%t0"), CallCallee: "1"}, "function symbol"},
		{"jmp with non-label", Instruction{Kind: InstructionJmp, TrueLabel: Immediate("1")}, "label operand"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := VerifyInstruction(tc.inst)
			if err == nil || !strings.Contains(err.Error(), tc.sub) {
				t.Fatalf("expected verifier error containing %q, got %v", tc.sub, err)
			}
		})
	}
}

func TestParseModule_VerifierRejectsInvalidKindsDeterministically(t *testing.T) {
	input := `.tac v1

func @bad() -> i32 {
.L0:
  %t0 = load 1
  ret %t0
}
`
	_, err := ParseModule(strings.NewReader(input))
	if err == nil || !strings.Contains(err.Error(), "stack slot pointer") {
		t.Fatalf("unexpected error: %v", err)
	}
}

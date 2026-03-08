package tac

import (
	"strings"
	"testing"
)

func TestValidateFunctionIR_ValidFallthrough(t *testing.T) {
	fn := Function{Name: "@ok", ReturnType: "i32"}
	fn.AddLabel(".L0")
	v := fn.AddInstruction(OpcodeConstI32, Immediate("1"))
	fn.AddLabel(".L1")
	fn.AddRet(v)

	if err := ValidateFunctionIR(fn); err != nil {
		t.Fatalf("validate: %v", err)
	}
}

func TestValidateFunctionIR_RejectsMissingSuccessorLabel(t *testing.T) {
	fn := Function{Name: "@bad", ReturnType: "i32"}
	fn.AddLabel(".L0")
	fn.AddJmp(".Lmissing")

	err := ValidateFunctionIR(fn)
	if err == nil || !strings.Contains(err.Error(), "undefined label") {
		t.Fatalf("expected undefined label error, got %v", err)
	}
}

func TestModuleAddFunction_ValidatesIR(t *testing.T) {
	fn := Function{Name: "@bad", ReturnType: "i32"}
	fn.AddLabel(".L0")
	fn.AddJmp(".Lmissing")

	var mod Module
	err := mod.AddFunction(fn)
	if err == nil || !strings.Contains(err.Error(), "undefined label") {
		t.Fatalf("expected undefined label error, got %v", err)
	}
}

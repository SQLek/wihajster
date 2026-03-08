package tac

import (
	"strings"
	"testing"
)

func TestEvaluateFunction_BasicArithmeticAndBranch(t *testing.T) {
	input := `.tac v1

func @sel(%a:i32, %b:i32) -> i32 {
.L0:
  %t0 = gt_s %a, %b
  br %t0, .L1, .L2
.L1:
  ret %a
.L2:
  ret %b
}
`
	mod, err := ParseModule(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse module: %v", err)
	}

	got, err := EvaluateFunction(mod, "@sel", []int32{3, 5}, EvalOptions{})
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if got != 5 {
		t.Fatalf("expected 5, got %d", got)
	}
}

func TestEvaluateFunction_MemoryAndCall(t *testing.T) {
	input := `.tac v1

func @inc(%x:i32) -> i32 {
.L0:
  %t0 = alloca i32
  store %t0, %x
  %t1 = load %t0
  %t2 = const.i32 1
  %t3 = add %t1, %t2
  ret %t3
}

func @main() -> i32 {
.L0:
  %t0 = const.i32 41
  %t1 = call @inc(%t0)
  ret %t1
}
`
	mod, err := ParseModule(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse module: %v", err)
	}

	got, err := EvaluateFunction(mod, "@main", nil, EvalOptions{})
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if got != 42 {
		t.Fatalf("expected 42, got %d", got)
	}
}

func TestEvaluateFunction_Recursion(t *testing.T) {
	input := `.tac v1

func @fact(%n:i32) -> i32 {
.L0:
  %t0 = const.i32 1
  %t1 = le_s %n, %t0
  br %t1, .L1, .L2
.L1:
  ret %t0
.L2:
  %t2 = const.i32 1
  %t3 = sub %n, %t2
  %t4 = call @fact(%t3)
  %t5 = mul %n, %t4
  ret %t5
}
`
	mod, err := ParseModule(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse module: %v", err)
	}

	got, err := EvaluateFunction(mod, "@fact", []int32{5}, EvalOptions{StepLimit: 50000, MaxCallDepth: 32})
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if got != 120 {
		t.Fatalf("expected 120, got %d", got)
	}
}

func TestEvaluateFunction_Errors(t *testing.T) {
	tests := []struct {
		name string
		mod  string
		fn   string
		args []int32
		msg  string
		opts EvalOptions
	}{
		{
			name: "missing function",
			mod: `.tac v1

func @main() -> i32 {
.L0:
  %t0 = const.i32 0
  ret %t0
}
`,
			fn:  "@none",
			msg: "missing function",
		},
		{
			name: "arity mismatch",
			mod: `.tac v1

func @f(%x:i32) -> i32 {
.L0:
  ret %x
}
`,
			fn:   "@f",
			args: []int32{1, 2},
			msg:  "expects 1 arguments, got 2",
		},
		{
			name: "divide by zero",
			mod: `.tac v1

func @f() -> i32 {
.L0:
  %t0 = const.i32 1
  %t1 = const.i32 0
  %t2 = div_s %t0, %t1
  ret %t2
}
`,
			fn:  "@f",
			msg: "division by zero",
		},
		{
			name: "unsupported opcode",
			mod: `.tac v1

func @f() -> i32 {
.L0:
  %t0 = phi %a, %b
  ret %t0
}
`,
			fn:  "@f",
			msg: "not enabled",
		},
		{
			name: "uninitialized load",
			mod: `.tac v1

func @f() -> i32 {
.L0:
  %t0 = alloca i32
  %t1 = load %t0
  ret %t1
}
`,
			fn:  "@f",
			msg: "uninitialized memory",
		},
		{
			name: "step limit",
			mod: `.tac v1

func @f() -> i32 {
.L0:
  jmp .L0
}
`,
			fn:   "@f",
			msg:  "step limit exceeded",
			opts: EvalOptions{StepLimit: 100},
		},
		{
			name: "missing ret",
			mod: `.tac v1

func @f() -> i32 {
.L0:
  %t0 = const.i32 1
}
`,
			fn:  "@f",
			msg: "has no terminator and no fallthrough successor",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mod, err := ParseModule(strings.NewReader(tc.mod))
			if err != nil {
				if strings.Contains(err.Error(), tc.msg) {
					return
				}
				t.Fatalf("parse module: %v", err)
			}
			_, err = EvaluateFunction(mod, tc.fn, tc.args, tc.opts)
			if err == nil {
				t.Fatalf("expected error containing %q", tc.msg)
			}
			if !strings.Contains(err.Error(), tc.msg) {
				t.Fatalf("expected error containing %q, got %v", tc.msg, err)
			}
		})
	}
}

func TestEvaluateFunction_ValidatesFunctionIR(t *testing.T) {
	mod := Module{Functions: []Function{{
		Name:       "@bad",
		ReturnType: "i32",
		Instructions: []Instruction{
			{Kind: InstructionLabel, Label: ".L0"},
			{Kind: InstructionJmp, TrueLabel: Label(".Lmissing")},
		},
	}}}

	_, err := EvaluateFunction(mod, "@bad", nil, EvalOptions{})
	if err == nil || !strings.Contains(err.Error(), "undefined label") {
		t.Fatalf("expected undefined label error, got %v", err)
	}
}

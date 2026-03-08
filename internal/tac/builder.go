package tac

import "fmt"

// NewTemp allocates a new deterministic temporary name for the function.
// Names are monotonically increasing: %t0, %t1, ...
func (f *Function) NewTemp() string {
	temp := fmt.Sprintf("%%t%d", f.nextTempID)
	f.nextTempID++
	return temp
}

// AddInstruction appends a value-producing operation instruction and returns
// the generated destination temporary.
func (f *Function) AddInstruction(opcode string, operands ...string) string {
	temp := f.NewTemp()
	f.Instructions = append(f.Instructions, Instruction{
		Kind:        InstructionOp,
		Destination: temp,
		Opcode:      opcode,
		Operands:    append([]string(nil), operands...),
	})
	return temp
}

// AddVoidInstruction appends a side-effect operation with no destination.
func (f *Function) AddVoidInstruction(opcode string, operands ...string) {
	f.Instructions = append(f.Instructions, Instruction{
		Kind:     InstructionOp,
		Opcode:   opcode,
		Operands: append([]string(nil), operands...),
	})
}

// AddCall emits a value-producing function call.
func (f *Function) AddCall(callee string, args ...string) string {
	temp := f.NewTemp()
	callArgs := make([]ValueRef, 0, len(args))
	for _, arg := range args {
		callArgs = append(callArgs, ValueRef(arg))
	}
	f.Instructions = append(f.Instructions, Instruction{
		Kind:        InstructionOp,
		Destination: temp,
		Opcode:      "call",
		CallCallee:  callee,
		CallArgs:    callArgs,
	})
	return temp
}

// AddCallVoid emits a call with ignored return value.
func (f *Function) AddCallVoid(callee string, args ...string) {
	callArgs := make([]ValueRef, 0, len(args))
	for _, arg := range args {
		callArgs = append(callArgs, ValueRef(arg))
	}
	f.Instructions = append(f.Instructions, Instruction{
		Kind:       InstructionOp,
		Opcode:     "call",
		CallCallee: callee,
		CallArgs:   callArgs,
	})
}

// AddLabel appends a label instruction.
func (f *Function) AddLabel(label string) {
	f.Instructions = append(f.Instructions, Instruction{Kind: InstructionLabel, Label: label})
}

// AddJmp appends an unconditional jump instruction.
func (f *Function) AddJmp(label string) {
	f.Instructions = append(f.Instructions, Instruction{Kind: InstructionJmp, Label: label})
}

// AddBr appends a conditional branch instruction.
func (f *Function) AddBr(condition, trueLabel, falseLabel string) {
	f.Instructions = append(f.Instructions, Instruction{
		Kind:       InstructionBr,
		Condition:  condition,
		TrueLabel:  trueLabel,
		FalseLabel: falseLabel,
	})
}

// AddRet appends a return instruction. Empty value means bare `ret`.
func (f *Function) AddRet(value string) {
	f.Instructions = append(f.Instructions, Instruction{Kind: InstructionRet, ReturnValue: value})
}

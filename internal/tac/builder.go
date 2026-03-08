package tac

import "fmt"

// NewTemp allocates a new deterministic temporary name for the function.
// Names are monotonically increasing: %t0, %t1, ...
func (f *Function) NewTemp() Operand {
	temp := Temp(fmt.Sprintf("%%t%d", f.nextTempID))
	f.nextTempID++
	return temp
}

func (f *Function) NewStackSlot() Operand {
	slot := StackSlotPointer(fmt.Sprintf("%%s%d", f.nextTempID))
	f.nextTempID++
	return slot
}

// AddInstruction appends a value-producing operation instruction and returns
// the generated destination temporary.
func (f *Function) AddInstruction(opcode Opcode, operands ...Operand) Operand {
	dst := f.NewTemp()
	if opcode == OpcodeAlloca {
		dst = f.NewStackSlot()
	}
	f.Instructions = append(f.Instructions, Instruction{
		Kind:           InstructionOp,
		HasDestination: true,
		Destination:    dst,
		Opcode:         opcode,
		Operands:       append([]Operand(nil), operands...),
	})
	return dst
}

// AddVoidInstruction appends a side-effect operation with no destination.
func (f *Function) AddVoidInstruction(opcode Opcode, operands ...Operand) {
	f.Instructions = append(f.Instructions, Instruction{
		Kind:     InstructionOp,
		Opcode:   opcode,
		Operands: append([]Operand(nil), operands...),
	})
}

// AddCall emits a value-producing function call.
func (f *Function) AddCall(callee Operand, args ...Operand) Operand {
	ops := append([]Operand{callee}, args...)
	return f.AddInstruction(OpcodeCall, ops...)
}

// AddCallVoid emits a call with ignored return value.
func (f *Function) AddCallVoid(callee Operand, args ...Operand) {
	ops := append([]Operand{callee}, args...)
	f.AddVoidInstruction(OpcodeCall, ops...)
}

// AddLabel appends a label instruction.
func (f *Function) AddLabel(label string) {
	f.Instructions = append(f.Instructions, Instruction{Kind: InstructionLabel, Label: label})
}

// AddJmp appends an unconditional jump instruction.
func (f *Function) AddJmp(label string) {
	f.Instructions = append(f.Instructions, Instruction{Kind: InstructionJmp, TrueLabel: Label(label)})
}

// AddBr appends a conditional branch instruction.
func (f *Function) AddBr(condition Operand, trueLabel, falseLabel string) {
	f.Instructions = append(f.Instructions, Instruction{
		Kind:       InstructionBr,
		Condition:  condition,
		TrueLabel:  Label(trueLabel),
		FalseLabel: Label(falseLabel),
	})
}

// AddRet appends a return instruction. Empty value means bare `ret`.
func (f *Function) AddRet(value Operand) {
	inst := Instruction{Kind: InstructionRet}
	if value.Kind != OperandInvalid {
		inst.HasReturnValue = true
		inst.ReturnValue = value
	}
	f.Instructions = append(f.Instructions, inst)
}

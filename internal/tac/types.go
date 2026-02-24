package tac

type Module struct {
	Functions []Function
}

type Function struct {
	Name       string
	Parameters []Parameter
	ReturnType string

	Instructions []Instruction
}

type Parameter struct {
	Name string
	Type string
}

type InstructionKind int

const (
	InstructionLabel InstructionKind = iota
	InstructionOp
	InstructionJmp
	InstructionBr
	InstructionRet
)

type Instruction struct {
	Kind InstructionKind

	Label string

	Destination string
	Opcode      string
	Operands    []string

	Condition  string
	TrueLabel  string
	FalseLabel string

	ReturnValue string
}

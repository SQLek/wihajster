package tac

import "fmt"

type Module struct {
	Functions []Function
}

type Function struct {
	Name       string
	Parameters []Parameter
	ReturnType string

	Instructions []Instruction

	nextTempID int
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

type Opcode int

const (
	OpcodeInvalid Opcode = iota
	OpcodeConstI32
	OpcodeConstI8
	OpcodeCopy
	OpcodeAdd
	OpcodeSub
	OpcodeMul
	OpcodeDivS
	OpcodeModS
	OpcodeAnd
	OpcodeOr
	OpcodeXor
	OpcodeShl
	OpcodeShrS
	OpcodeEq
	OpcodeNe
	OpcodeLtS
	OpcodeLeS
	OpcodeGtS
	OpcodeGeS
	OpcodeNeg
	OpcodeNot
	OpcodeLogicNot
	OpcodeCall
	OpcodeAlloca
	OpcodeLoad
	OpcodeStore
)

var opcodeNames = map[Opcode]string{
	OpcodeConstI32: "const.i32", OpcodeConstI8: "const.i8", OpcodeCopy: "copy",
	OpcodeAdd: "add", OpcodeSub: "sub", OpcodeMul: "mul", OpcodeDivS: "div_s", OpcodeModS: "mod_s",
	OpcodeAnd: "and", OpcodeOr: "or", OpcodeXor: "xor", OpcodeShl: "shl", OpcodeShrS: "shr_s",
	OpcodeEq: "eq", OpcodeNe: "ne", OpcodeLtS: "lt_s", OpcodeLeS: "le_s", OpcodeGtS: "gt_s", OpcodeGeS: "ge_s",
	OpcodeNeg: "neg", OpcodeNot: "not", OpcodeLogicNot: "logic_not", OpcodeCall: "call", OpcodeAlloca: "alloca", OpcodeLoad: "load", OpcodeStore: "store",
}

var coreOpcodeByName = map[string]Opcode{}

func init() {
	for op, name := range opcodeNames {
		coreOpcodeByName[name] = op
	}
}

func ParseOpcode(name string) (Opcode, bool, bool) {
	if op, ok := coreOpcodeByName[name]; ok {
		return op, true, false
	}
	switch name {
	case "gep", "zext", "sext", "trunc", "bitcast", "phi":
		return OpcodeInvalid, false, true
	default:
		return OpcodeInvalid, false, false
	}
}

func (o Opcode) String() string {
	if name, ok := opcodeNames[o]; ok {
		return name
	}
	return "invalid"
}

type OperandKind int

const (
	OperandInvalid OperandKind = iota
	OperandTemp
	OperandParam
	OperandImmediate
	OperandLabel
	OperandFunctionSymbol
	OperandStackSlotPointer
)

type Operand struct {
	Kind OperandKind
	Text string
}

func (o Operand) String() string { return o.Text }

func Temp(name string) Operand             { return Operand{Kind: OperandTemp, Text: name} }
func Param(name string) Operand            { return Operand{Kind: OperandParam, Text: name} }
func Immediate(value string) Operand       { return Operand{Kind: OperandImmediate, Text: value} }
func Label(name string) Operand            { return Operand{Kind: OperandLabel, Text: name} }
func FunctionSymbol(name string) Operand   { return Operand{Kind: OperandFunctionSymbol, Text: name} }
func StackSlotPointer(name string) Operand { return Operand{Kind: OperandStackSlotPointer, Text: name} }

type Instruction struct {
	Kind InstructionKind

	Label string

	HasDestination bool
	Destination    Operand
	Opcode         Opcode
	Operands       []Operand

	Condition  Operand
	TrueLabel  Operand
	FalseLabel Operand

	HasReturnValue bool
	ReturnValue    Operand
}

func VerifyInstruction(inst Instruction) error {
	switch inst.Kind {
	case InstructionLabel:
		if inst.Label == "" {
			return fmt.Errorf("label instruction requires a label")
		}
		return nil
	case InstructionJmp:
		if inst.TrueLabel.Kind != OperandLabel {
			return fmt.Errorf("jmp target must be a label operand")
		}
		return nil
	case InstructionBr:
		if inst.Condition.Kind == OperandInvalid {
			return fmt.Errorf("br condition must be set")
		}
		if inst.TrueLabel.Kind != OperandLabel || inst.FalseLabel.Kind != OperandLabel {
			return fmt.Errorf("br targets must be label operands")
		}
		return nil
	case InstructionRet:
		if inst.HasReturnValue && inst.ReturnValue.Kind == OperandInvalid {
			return fmt.Errorf("ret value kind is invalid")
		}
		return nil
	case InstructionOp:
		if inst.Opcode == OpcodeInvalid {
			return fmt.Errorf("operation requires a valid opcode")
		}
		return verifyOpcodeOperands(inst)
	default:
		return fmt.Errorf("unsupported instruction kind %d", inst.Kind)
	}
}

func verifyOpcodeOperands(inst Instruction) error {
	requireKinds := func(kinds ...OperandKind) error {
		if len(inst.Operands) != len(kinds) {
			return fmt.Errorf("opcode %s expects %d operands, got %d", inst.Opcode, len(kinds), len(inst.Operands))
		}
		for i, want := range kinds {
			if inst.Operands[i].Kind != want {
				return fmt.Errorf("opcode %s operand %d must be %s, got %s", inst.Opcode, i+1, operandKindName(want), operandKindName(inst.Operands[i].Kind))
			}
		}
		return nil
	}
	valueKind := func(k OperandKind) bool { return k == OperandTemp || k == OperandParam || k == OperandImmediate }

	switch inst.Opcode {
	case OpcodeConstI32, OpcodeConstI8:
		return requireKinds(OperandImmediate)
	case OpcodeCopy:
		if len(inst.Operands) != 1 || !valueKind(inst.Operands[0].Kind) {
			return fmt.Errorf("opcode %s operand 1 must be a value", inst.Opcode)
		}
	case OpcodeAdd, OpcodeSub, OpcodeMul, OpcodeDivS, OpcodeModS, OpcodeAnd, OpcodeOr, OpcodeXor, OpcodeShl, OpcodeShrS, OpcodeEq, OpcodeNe, OpcodeLtS, OpcodeLeS, OpcodeGtS, OpcodeGeS:
		if len(inst.Operands) != 2 || !valueKind(inst.Operands[0].Kind) || !valueKind(inst.Operands[1].Kind) {
			return fmt.Errorf("opcode %s expects two value operands", inst.Opcode)
		}
	case OpcodeNeg, OpcodeNot, OpcodeLogicNot:
		if len(inst.Operands) != 1 || !valueKind(inst.Operands[0].Kind) {
			return fmt.Errorf("opcode %s expects one value operand", inst.Opcode)
		}
	case OpcodeAlloca:
		return requireKinds(OperandImmediate)
	case OpcodeLoad:
		if len(inst.Operands) != 1 || inst.Operands[0].Kind != OperandStackSlotPointer {
			return fmt.Errorf("opcode load expects one stack slot pointer operand")
		}
	case OpcodeStore:
		if len(inst.Operands) != 2 || inst.Operands[0].Kind != OperandStackSlotPointer || !valueKind(inst.Operands[1].Kind) {
			return fmt.Errorf("opcode store expects stack slot pointer and value operands")
		}
	case OpcodeCall:
		if len(inst.Operands) == 0 || inst.Operands[0].Kind != OperandFunctionSymbol {
			return fmt.Errorf("opcode call first operand must be function symbol")
		}
		for i := 1; i < len(inst.Operands); i++ {
			if !valueKind(inst.Operands[i].Kind) {
				return fmt.Errorf("opcode call argument %d must be value operand", i)
			}
		}
	default:
		return fmt.Errorf("unsupported opcode %s", inst.Opcode)
	}
	if inst.HasDestination && inst.Destination.Kind != OperandTemp && inst.Destination.Kind != OperandStackSlotPointer {
		return fmt.Errorf("destination must be temp or stack slot pointer")
	}
	return nil
}

func operandKindName(k OperandKind) string {
	switch k {
	case OperandTemp:
		return "temp"
	case OperandParam:
		return "param"
	case OperandImmediate:
		return "immediate"
	case OperandLabel:
		return "label"
	case OperandFunctionSymbol:
		return "function symbol"
	case OperandStackSlotPointer:
		return "stack slot pointer"
	default:
		return "invalid"
	}
}

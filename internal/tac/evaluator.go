package tac

import (
	"fmt"
	"strconv"
	"strings"
)

type EvalOptions struct {
	StepLimit    int
	MaxCallDepth int
}

const (
	defaultStepLimit    = 100000
	defaultMaxCallDepth = 128
)

type runtimeValueKind int

const (
	valueI32 runtimeValueKind = iota
	valuePtr
)

type runtimeValue struct {
	kind runtimeValueKind
	i32  int32
	ptr  int
}

type memoryCell struct {
	value       runtimeValue
	initialized bool
}

type evalState struct {
	mod          Module
	funcs        map[string]Function
	steps        int
	stepLimit    int
	maxCallDepth int
}

type evalFrame struct {
	fn       Function
	values   map[string]runtimeValue
	memory   map[int]memoryCell
	nextAddr int
	labels   map[string]int
}

func EvaluateFunction(mod Module, functionName string, args []int32, opts EvalOptions) (int32, error) {
	if opts.StepLimit <= 0 {
		opts.StepLimit = defaultStepLimit
	}
	if opts.MaxCallDepth <= 0 {
		opts.MaxCallDepth = defaultMaxCallDepth
	}

	funcs := make(map[string]Function, len(mod.Functions))
	for _, fn := range mod.Functions {
		if err := ValidateFunctionIR(fn); err != nil {
			return 0, err
		}
		funcs[fn.Name] = fn
	}

	st := &evalState{
		mod:          mod,
		funcs:        funcs,
		stepLimit:    opts.StepLimit,
		maxCallDepth: opts.MaxCallDepth,
	}

	callArgs := make([]runtimeValue, 0, len(args))
	for _, a := range args {
		callArgs = append(callArgs, runtimeValue{kind: valueI32, i32: a})
	}
	ret, err := st.evalCall(functionName, callArgs, 0)
	if err != nil {
		return 0, err
	}
	if ret.kind != valueI32 {
		return 0, fmt.Errorf("function %s returned non-i32 value", functionName)
	}
	return ret.i32, nil
}

func (s *evalState) evalCall(functionName string, args []runtimeValue, depth int) (runtimeValue, error) {
	if depth >= s.maxCallDepth {
		return runtimeValue{}, fmt.Errorf("maximum call depth exceeded at %s", functionName)
	}

	fn, ok := s.funcs[functionName]
	if !ok {
		return runtimeValue{}, fmt.Errorf("missing function %s", functionName)
	}
	if len(args) != len(fn.Parameters) {
		return runtimeValue{}, fmt.Errorf("function %s expects %d arguments, got %d", functionName, len(fn.Parameters), len(args))
	}

	frame := evalFrame{
		fn:     fn,
		values: map[string]runtimeValue{},
		memory: map[int]memoryCell{},
		labels: map[string]int{},
	}

	for i, p := range fn.Parameters {
		frame.values[p.Name] = args[i]
	}
	for i, inst := range fn.Instructions {
		if inst.Kind == InstructionLabel {
			frame.labels[inst.Label] = i
		}
	}

	pc := 0
	for pc < len(fn.Instructions) {
		s.steps++
		if s.steps > s.stepLimit {
			return runtimeValue{}, fmt.Errorf("step limit exceeded while evaluating %s", functionName)
		}

		inst := fn.Instructions[pc]
		switch inst.Kind {
		case InstructionLabel:
			pc++
		case InstructionJmp:
			next, ok := frame.labels[inst.TrueLabel.Text]
			if !ok {
				return runtimeValue{}, fmt.Errorf("invalid jump label %s in %s", inst.TrueLabel.Text, functionName)
			}
			pc = next
		case InstructionBr:
			cond, err := frame.resolveI32(inst.Condition.Text)
			if err != nil {
				return runtimeValue{}, err
			}
			target := inst.FalseLabel.Text
			if cond != 0 {
				target = inst.TrueLabel.Text
			}
			next, ok := frame.labels[target]
			if !ok {
				return runtimeValue{}, fmt.Errorf("invalid branch label %s in %s", target, functionName)
			}
			pc = next
		case InstructionRet:
			if !inst.HasReturnValue {
				return runtimeValue{kind: valueI32, i32: 0}, nil
			}
			v, err := frame.resolveValue(inst.ReturnValue.Text)
			if err != nil {
				return runtimeValue{}, err
			}
			return v, nil
		case InstructionOp:
			res, hasResult, err := s.evalOp(&frame, inst, depth)
			if err != nil {
				return runtimeValue{}, err
			}
			if hasResult {
				if !inst.HasDestination {
					return runtimeValue{}, fmt.Errorf("opcode %s produced value without destination in %s", inst.Opcode, functionName)
				}
				frame.values[inst.Destination.Text] = res
			}
			pc++
		default:
			return runtimeValue{}, fmt.Errorf("unsupported instruction kind %d in %s", inst.Kind, functionName)
		}
	}

	return runtimeValue{}, fmt.Errorf("function %s ended without ret", functionName)
}

func (s *evalState) evalOp(frame *evalFrame, inst Instruction, depth int) (runtimeValue, bool, error) {
	op := inst.Opcode
	ops := inst.Operands

	needCount := func(n int) error {
		if len(ops) != n {
			return fmt.Errorf("opcode %s expects %d operands, got %d", op, n, len(ops))
		}
		return nil
	}

	switch op {
	case OpcodeConstI32:
		if err := needCount(1); err != nil {
			return runtimeValue{}, false, fmt.Errorf("opcode call expects at least one operand")
		}
		n, err := strconv.ParseInt(strings.TrimSpace(ops[0].Text), 10, 32)
		if err != nil {
			return runtimeValue{}, false, fmt.Errorf("invalid const.i32 operand %q", ops[0].Text)
		}
		return runtimeValue{kind: valueI32, i32: int32(n)}, true, nil
	case OpcodeConstI8:
		if err := needCount(1); err != nil {
			return runtimeValue{}, false, err
		}
		n, err := strconv.ParseInt(strings.TrimSpace(ops[0].Text), 10, 8)
		if err != nil {
			return runtimeValue{}, false, fmt.Errorf("invalid const.i8 operand %q", ops[0].Text)
		}
		return runtimeValue{kind: valueI32, i32: int32(int8(n))}, true, nil
	case OpcodeCopy:
		if err := needCount(1); err != nil {
			return runtimeValue{}, false, err
		}
		v, err := frame.resolveValue(ops[0].Text)
		if err != nil {
			return runtimeValue{}, false, err
		}
		return v, true, nil
	case OpcodeAlloca:
		if err := needCount(1); err != nil {
			return runtimeValue{}, false, err
		}
		addr := frame.nextAddr
		frame.nextAddr++
		frame.memory[addr] = memoryCell{}
		return runtimeValue{kind: valuePtr, ptr: addr}, true, nil
	case OpcodeLoad:
		if err := needCount(1); err != nil {
			return runtimeValue{}, false, err
		}
		ptr, err := frame.resolvePtr(ops[0].Text)
		if err != nil {
			return runtimeValue{}, false, err
		}
		cell, ok := frame.memory[ptr]
		if !ok || !cell.initialized {
			return runtimeValue{}, false, fmt.Errorf("load from uninitialized memory at %d", ptr)
		}
		return cell.value, true, nil
	case OpcodeLoadIndirect:
		if err := needCount(1); err != nil {
			return runtimeValue{}, false, err
		}
		ptr, err := frame.resolvePtr(ops[0].Text)
		if err != nil {
			return runtimeValue{}, false, err
		}
		cell, ok := frame.memory[ptr]
		if !ok || !cell.initialized {
			return runtimeValue{}, false, fmt.Errorf("load from uninitialized memory at %d", ptr)
		}
		return cell.value, true, nil
	case OpcodeStore:
		if err := needCount(2); err != nil {
			return runtimeValue{}, false, err
		}
		ptr, err := frame.resolvePtr(ops[0].Text)
		if err != nil {
			return runtimeValue{}, false, err
		}
		val, err := frame.resolveValue(ops[1].Text)
		if err != nil {
			return runtimeValue{}, false, err
		}
		frame.memory[ptr] = memoryCell{value: val, initialized: true}
		return runtimeValue{}, false, nil
	case OpcodeStoreIndirect:
		if err := needCount(2); err != nil {
			return runtimeValue{}, false, err
		}
		ptr, err := frame.resolvePtr(ops[0].Text)
		if err != nil {
			return runtimeValue{}, false, err
		}
		val, err := frame.resolveValue(ops[1].Text)
		if err != nil {
			return runtimeValue{}, false, err
		}
		frame.memory[ptr] = memoryCell{value: val, initialized: true}
		return runtimeValue{}, false, nil
	case OpcodeCall:
		if inst.CallCallee == "" {
			return runtimeValue{}, false, fmt.Errorf("opcode call requires call callee")
		}
		argv := make([]runtimeValue, 0, len(inst.CallArgs))
		for _, a := range inst.CallArgs {
			v, err := frame.resolveValue(a.Text)
			if err != nil {
				return runtimeValue{}, false, err
			}
			argv = append(argv, v)
		}
		ret, err := s.evalCall(inst.CallCallee, argv, depth+1)
		if err != nil {
			return runtimeValue{}, false, err
		}
		if !inst.HasDestination {
			return runtimeValue{}, false, nil
		}
		return ret, true, nil
	case OpcodeNeg, OpcodeNot, OpcodeLogicNot:
		if err := needCount(1); err != nil {
			return runtimeValue{}, false, err
		}
		a, err := frame.resolveI32(ops[0].Text)
		if err != nil {
			return runtimeValue{}, false, err
		}
		switch op {
		case OpcodeNeg:
			return runtimeValue{kind: valueI32, i32: -a}, true, nil
		case OpcodeNot:
			return runtimeValue{kind: valueI32, i32: ^a}, true, nil
		default:
			if a == 0 {
				return runtimeValue{kind: valueI32, i32: 1}, true, nil
			}
			return runtimeValue{kind: valueI32, i32: 0}, true, nil
		}
	case OpcodeAdd, OpcodeSub, OpcodeMul, OpcodeDivS, OpcodeModS, OpcodeAnd, OpcodeOr, OpcodeXor, OpcodeShl, OpcodeShrS, OpcodeEq, OpcodeNe, OpcodeLtS, OpcodeLeS, OpcodeGtS, OpcodeGeS:
		if err := needCount(2); err != nil {
			return runtimeValue{}, false, err
		}
		a, err := frame.resolveI32(ops[0].Text)
		if err != nil {
			return runtimeValue{}, false, err
		}
		b, err := frame.resolveI32(ops[1].Text)
		if err != nil {
			return runtimeValue{}, false, err
		}
		switch op {
		case OpcodeAdd:
			return runtimeValue{kind: valueI32, i32: a + b}, true, nil
		case OpcodeSub:
			return runtimeValue{kind: valueI32, i32: a - b}, true, nil
		case OpcodeMul:
			return runtimeValue{kind: valueI32, i32: a * b}, true, nil
		case OpcodeDivS:
			if b == 0 {
				return runtimeValue{}, false, fmt.Errorf("division by zero")
			}
			return runtimeValue{kind: valueI32, i32: a / b}, true, nil
		case OpcodeModS:
			if b == 0 {
				return runtimeValue{}, false, fmt.Errorf("modulo by zero")
			}
			return runtimeValue{kind: valueI32, i32: a % b}, true, nil
		case OpcodeAnd:
			return runtimeValue{kind: valueI32, i32: a & b}, true, nil
		case OpcodeOr:
			return runtimeValue{kind: valueI32, i32: a | b}, true, nil
		case OpcodeXor:
			return runtimeValue{kind: valueI32, i32: a ^ b}, true, nil
		case OpcodeShl:
			return runtimeValue{kind: valueI32, i32: a << uint32(b)}, true, nil
		case OpcodeShrS:
			return runtimeValue{kind: valueI32, i32: a >> uint32(b)}, true, nil
		case OpcodeEq:
			return boolI32(a == b), true, nil
		case OpcodeNe:
			return boolI32(a != b), true, nil
		case OpcodeLtS:
			return boolI32(a < b), true, nil
		case OpcodeLeS:
			return boolI32(a <= b), true, nil
		case OpcodeGtS:
			return boolI32(a > b), true, nil
		default:
			return boolI32(a >= b), true, nil
		}
	default:
		return runtimeValue{}, false, fmt.Errorf("opcode %s not supported by evaluator v1", op)
	}
}

func boolI32(v bool) runtimeValue {
	if v {
		return runtimeValue{kind: valueI32, i32: 1}
	}
	return runtimeValue{kind: valueI32, i32: 0}
}

func (f *evalFrame) resolveValue(token string) (runtimeValue, error) {
	token = strings.TrimSpace(token)
	if v, ok := f.values[token]; ok {
		return v, nil
	}
	n, err := strconv.ParseInt(token, 10, 32)
	if err == nil {
		return runtimeValue{kind: valueI32, i32: int32(n)}, nil
	}
	return runtimeValue{}, fmt.Errorf("unknown value %s", token)
}

func (f *evalFrame) resolveI32(token string) (int32, error) {
	v, err := f.resolveValue(token)
	if err != nil {
		return 0, err
	}
	if v.kind != valueI32 {
		return 0, fmt.Errorf("expected i32 value, got pointer")
	}
	return v.i32, nil
}

func (f *evalFrame) resolvePtr(token string) (int, error) {
	v, err := f.resolveValue(token)
	if err != nil {
		return 0, err
	}
	if v.kind != valuePtr {
		return 0, fmt.Errorf("expected pointer value")
	}
	return v.ptr, nil
}

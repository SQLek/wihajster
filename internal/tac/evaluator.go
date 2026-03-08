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
			next, ok := frame.labels[inst.Label]
			if !ok {
				return runtimeValue{}, fmt.Errorf("invalid jump label %s in %s", inst.Label, functionName)
			}
			pc = next
		case InstructionBr:
			cond, err := frame.resolveI32(inst.Condition)
			if err != nil {
				return runtimeValue{}, err
			}
			target := inst.FalseLabel
			if cond != 0 {
				target = inst.TrueLabel
			}
			next, ok := frame.labels[target]
			if !ok {
				return runtimeValue{}, fmt.Errorf("invalid branch label %s in %s", target, functionName)
			}
			pc = next
		case InstructionRet:
			if inst.ReturnValue == "" {
				return runtimeValue{kind: valueI32, i32: 0}, nil
			}
			v, err := frame.resolveValue(inst.ReturnValue)
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
				if inst.Destination == "" {
					return runtimeValue{}, fmt.Errorf("opcode %s produced value without destination in %s", inst.Opcode, functionName)
				}
				frame.values[inst.Destination] = res
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
	case "const.i32":
		if err := needCount(1); err != nil {
			return runtimeValue{}, false, err
		}
		n, err := strconv.ParseInt(strings.TrimSpace(ops[0]), 10, 32)
		if err != nil {
			return runtimeValue{}, false, fmt.Errorf("invalid const.i32 operand %q", ops[0])
		}
		return runtimeValue{kind: valueI32, i32: int32(n)}, true, nil
	case "const.i8":
		if err := needCount(1); err != nil {
			return runtimeValue{}, false, err
		}
		n, err := strconv.ParseInt(strings.TrimSpace(ops[0]), 10, 8)
		if err != nil {
			return runtimeValue{}, false, fmt.Errorf("invalid const.i8 operand %q", ops[0])
		}
		return runtimeValue{kind: valueI32, i32: int32(int8(n))}, true, nil
	case "copy":
		if err := needCount(1); err != nil {
			return runtimeValue{}, false, err
		}
		v, err := frame.resolveValue(ops[0])
		if err != nil {
			return runtimeValue{}, false, err
		}
		return v, true, nil
	case "alloca":
		if err := needCount(1); err != nil {
			return runtimeValue{}, false, err
		}
		addr := frame.nextAddr
		frame.nextAddr++
		frame.memory[addr] = memoryCell{}
		return runtimeValue{kind: valuePtr, ptr: addr}, true, nil
	case "load":
		if err := needCount(1); err != nil {
			return runtimeValue{}, false, err
		}
		ptr, err := frame.resolvePtr(ops[0])
		if err != nil {
			return runtimeValue{}, false, err
		}
		cell, ok := frame.memory[ptr]
		if !ok || !cell.initialized {
			return runtimeValue{}, false, fmt.Errorf("load from uninitialized memory at %d", ptr)
		}
		return cell.value, true, nil
	case "store":
		if err := needCount(2); err != nil {
			return runtimeValue{}, false, err
		}
		ptr, err := frame.resolvePtr(ops[0])
		if err != nil {
			return runtimeValue{}, false, err
		}
		val, err := frame.resolveValue(ops[1])
		if err != nil {
			return runtimeValue{}, false, err
		}
		frame.memory[ptr] = memoryCell{value: val, initialized: true}
		return runtimeValue{}, false, nil
	case "call":
		callee := inst.CallCallee
		callArgs := inst.CallArgs
		if callee == "" {
			if len(ops) != 1 {
				return runtimeValue{}, false, fmt.Errorf("opcode call expects 1 operand, got %d", len(ops))
			}
			parsedCallee, parsedArgs, err := parseCallText(ops[0])
			if err != nil {
				return runtimeValue{}, false, err
			}
			callee = parsedCallee
			callArgs = parsedArgs
		}
		argv := make([]runtimeValue, 0, len(callArgs))
		for _, a := range callArgs {
			v, err := frame.resolveValue(string(a))
			if err != nil {
				return runtimeValue{}, false, err
			}
			argv = append(argv, v)
		}
		ret, err := s.evalCall(callee, argv, depth+1)
		if err != nil {
			return runtimeValue{}, false, err
		}
		if inst.Destination == "" {
			return runtimeValue{}, false, nil
		}
		return ret, true, nil
	case "neg", "not", "logic_not":
		if err := needCount(1); err != nil {
			return runtimeValue{}, false, err
		}
		a, err := frame.resolveI32(ops[0])
		if err != nil {
			return runtimeValue{}, false, err
		}
		switch op {
		case "neg":
			return runtimeValue{kind: valueI32, i32: -a}, true, nil
		case "not":
			return runtimeValue{kind: valueI32, i32: ^a}, true, nil
		default:
			if a == 0 {
				return runtimeValue{kind: valueI32, i32: 1}, true, nil
			}
			return runtimeValue{kind: valueI32, i32: 0}, true, nil
		}
	case "add", "sub", "mul", "div_s", "mod_s", "and", "or", "xor", "shl", "shr_s", "eq", "ne", "lt_s", "le_s", "gt_s", "ge_s":
		if err := needCount(2); err != nil {
			return runtimeValue{}, false, err
		}
		a, err := frame.resolveI32(ops[0])
		if err != nil {
			return runtimeValue{}, false, err
		}
		b, err := frame.resolveI32(ops[1])
		if err != nil {
			return runtimeValue{}, false, err
		}
		switch op {
		case "add":
			return runtimeValue{kind: valueI32, i32: a + b}, true, nil
		case "sub":
			return runtimeValue{kind: valueI32, i32: a - b}, true, nil
		case "mul":
			return runtimeValue{kind: valueI32, i32: a * b}, true, nil
		case "div_s":
			if b == 0 {
				return runtimeValue{}, false, fmt.Errorf("division by zero")
			}
			return runtimeValue{kind: valueI32, i32: a / b}, true, nil
		case "mod_s":
			if b == 0 {
				return runtimeValue{}, false, fmt.Errorf("modulo by zero")
			}
			return runtimeValue{kind: valueI32, i32: a % b}, true, nil
		case "and":
			return runtimeValue{kind: valueI32, i32: a & b}, true, nil
		case "or":
			return runtimeValue{kind: valueI32, i32: a | b}, true, nil
		case "xor":
			return runtimeValue{kind: valueI32, i32: a ^ b}, true, nil
		case "shl":
			return runtimeValue{kind: valueI32, i32: a << uint32(b)}, true, nil
		case "shr_s":
			return runtimeValue{kind: valueI32, i32: a >> uint32(b)}, true, nil
		case "eq":
			return boolI32(a == b), true, nil
		case "ne":
			return boolI32(a != b), true, nil
		case "lt_s":
			return boolI32(a < b), true, nil
		case "le_s":
			return boolI32(a <= b), true, nil
		case "gt_s":
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

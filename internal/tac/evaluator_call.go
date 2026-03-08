package tac

import "fmt"

func (s *evalState) evalCallInstruction(frame *evalFrame, inst Instruction, depth int) (runtimeValue, bool, error) {
	callee := inst.CallCallee
	callArgs := inst.CallArgs
	if callee == "" {
		if len(inst.Operands) != 1 {
			return runtimeValue{}, false, fmt.Errorf("opcode call expects 1 operand, got %d", len(inst.Operands))
		}
		parsedCallee, parsedArgs, err := parseCallText(inst.Operands[0])
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
}

package tac

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

func WriteModule(w io.Writer, mod Module) error {
	bw := bufio.NewWriter(w)
	if _, err := bw.WriteString(".tac v1\n\n"); err != nil {
		return err
	}

	for i, fn := range mod.Functions {
		if err := writeFunction(bw, fn); err != nil {
			return err
		}
		if i < len(mod.Functions)-1 {
			if _, err := bw.WriteString("\n"); err != nil {
				return err
			}
		}
	}

	return bw.Flush()
}

func writeFunction(w io.Writer, fn Function) error {
	if _, err := fmt.Fprintf(w, "func %s(%s) -> %s {\n", fn.Name, formatParams(fn.Parameters), fn.ReturnType); err != nil {
		return err
	}

	for _, inst := range fn.Instructions {
		line, err := formatInstruction(inst)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "  %s\n", line); err != nil {
			return err
		}
	}

	_, err := io.WriteString(w, "}\n")
	return err
}

func formatParams(params []Parameter) string {
	parts := make([]string, 0, len(params))
	for _, p := range params {
		parts = append(parts, fmt.Sprintf("%s:%s", p.Name, p.Type))
	}
	return strings.Join(parts, ", ")
}

func formatInstruction(inst Instruction) (string, error) {
	switch inst.Kind {
	case InstructionLabel:
		return inst.Label + ":", nil
	case InstructionJmp:
		return "jmp " + inst.Label, nil
	case InstructionBr:
		return fmt.Sprintf("br %s, %s, %s", inst.Condition, inst.TrueLabel, inst.FalseLabel), nil
	case InstructionRet:
		if inst.ReturnValue == "" {
			return "ret", nil
		}
		return "ret " + inst.ReturnValue, nil
	case InstructionOp:
		line := inst.Opcode
		if len(inst.Operands) > 0 {
			line += " " + strings.Join(inst.Operands, ", ")
		}
		if inst.Destination != "" {
			line = fmt.Sprintf("%s = %s", inst.Destination, line)
		}
		return line, nil
	default:
		return "", fmt.Errorf("unsupported instruction kind: %d", inst.Kind)
	}
}

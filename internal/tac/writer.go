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
		if err := VerifyInstruction(inst); err != nil {
			return err
		}
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
		return "jmp " + inst.TrueLabel.Text, nil
	case InstructionBr:
		return fmt.Sprintf("br %s, %s, %s", inst.Condition.Text, inst.TrueLabel.Text, inst.FalseLabel.Text), nil
	case InstructionRet:
		if !inst.HasReturnValue {
			return "ret", nil
		}
		return "ret " + inst.ReturnValue.Text, nil
	case InstructionOp:
		line := inst.Opcode.String()
		if len(inst.Operands) > 0 {
			if inst.Opcode == OpcodeCall {
				line += " " + formatCallInstructionOperands(inst.Operands)
			} else {
				line += " " + formatOperands(inst.Operands)
			}
		}
		if inst.HasDestination {
			line = fmt.Sprintf("%s = %s", inst.Destination.Text, line)
		}
		return line, nil
	default:
		return "", fmt.Errorf("unsupported instruction kind: %d", inst.Kind)
	}
}

func formatCallInstructionOperands(ops []Operand) string {
	callee := ops[0].Text
	if len(ops) == 1 {
		return callee + "()"
	}
	args := make([]string, 0, len(ops)-1)
	for _, op := range ops[1:] {
		args = append(args, op.Text)
	}
	return callee + "(" + strings.Join(args, ", ") + ")"
}

func formatOperands(ops []Operand) string {
	parts := make([]string, 0, len(ops))
	for _, op := range ops {
		parts = append(parts, op.Text)
	}
	return strings.Join(parts, ", ")
}

package tac

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

var coreOpcodes = map[string]struct{}{
	"const.i32": {}, "const.i8": {}, "copy": {},
	"add": {}, "sub": {}, "mul": {}, "div_s": {}, "mod_s": {},
	"and": {}, "or": {}, "xor": {}, "shl": {}, "shr_s": {},
	"eq": {}, "ne": {}, "lt_s": {}, "le_s": {}, "gt_s": {}, "ge_s": {},
	"neg": {}, "not": {}, "logic_not": {}, "call": {},
}

var optionalOpcodes = map[string]struct{}{
	"alloca": {}, "load": {}, "store": {}, "gep": {},
	"zext": {}, "sext": {}, "trunc": {}, "bitcast": {}, "phi": {},
}

func ParseModule(r io.Reader) (Module, error) {
	p := parser{reader: bufio.NewReader(r)}
	return p.parse()
}

type parser struct {
	reader *bufio.Reader
	line   int
}

func (p *parser) parse() (Module, error) {
	var mod Module

	headerSeen := false
	funcNames := map[string]struct{}{}

	for {
		line, err := p.nextLogicalLine()
		if err == io.EOF {
			break
		}
		if err != nil {
			return Module{}, err
		}

		if !headerSeen {
			if line != ".tac v1" {
				return Module{}, p.errf("missing or invalid header, expected '.tac v1'")
			}
			headerSeen = true
			continue
		}

		if strings.HasPrefix(line, ".meta ") {
			continue
		}
		if line == ".tac v1" {
			return Module{}, p.errf("duplicated header")
		}
		if !strings.HasPrefix(line, "func ") {
			return Module{}, p.errf("unexpected line outside function: %q", line)
		}

		fn, err := p.parseFunction(line)
		if err != nil {
			return Module{}, err
		}
		if _, exists := funcNames[fn.Name]; exists {
			return Module{}, p.errf("function %q defined multiple times", fn.Name)
		}
		funcNames[fn.Name] = struct{}{}
		mod.Functions = append(mod.Functions, fn)
	}

	if !headerSeen {
		return Module{}, p.errf("missing header")
	}

	return mod, nil
}

func (p *parser) parseFunction(header string) (Function, error) {
	fn, err := parseFunctionHeader(header)
	if err != nil {
		return Function{}, p.errf("%v", err)
	}

	definedLabels := map[string]struct{}{}
	usedLabels := map[string]struct{}{}
	definedDestinations := map[string]struct{}{}

	for {
		line, err := p.nextLogicalLine()
		if err != nil {
			if err == io.EOF {
				return Function{}, p.errf("function %q is missing closing brace", fn.Name)
			}
			return Function{}, err
		}

		if line == "}" {
			break
		}

		inst, err := parseInstruction(line)
		if err != nil {
			return Function{}, p.errf("%v", err)
		}

		if inst.Kind == InstructionLabel {
			if _, exists := definedLabels[inst.Label]; exists {
				return Function{}, p.errf("label %q defined multiple times", inst.Label)
			}
			definedLabels[inst.Label] = struct{}{}
		}
		if inst.Destination != "" {
			if _, exists := definedDestinations[inst.Destination]; exists {
				return Function{}, p.errf("destination %q redefined", inst.Destination)
			}
			definedDestinations[inst.Destination] = struct{}{}
		}
		switch inst.Kind {
		case InstructionJmp:
			usedLabels[inst.Label] = struct{}{}
		case InstructionBr:
			usedLabels[inst.TrueLabel] = struct{}{}
			usedLabels[inst.FalseLabel] = struct{}{}
		}

		fn.Instructions = append(fn.Instructions, inst)
	}

	for label := range usedLabels {
		if _, exists := definedLabels[label]; !exists {
			return Function{}, p.errf("label %q is referenced but not defined", label)
		}
	}

	return fn, nil
}

func (p *parser) nextLogicalLine() (string, error) {
	for {
		line, err := p.reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return "", err
		}
		if err == io.EOF && len(line) == 0 {
			return "", io.EOF
		}

		p.line++
		line = strings.TrimRight(line, "\r\n")
		line = stripComment(line)
		line = strings.TrimSpace(line)
		if line == "" {
			if err == io.EOF {
				return "", io.EOF
			}
			continue
		}
		return line, nil
	}
}

func parseFunctionHeader(line string) (Function, error) {
	if !strings.HasPrefix(line, "func ") || !strings.HasSuffix(line, "{") {
		return Function{}, fmt.Errorf("invalid function header: %q", line)
	}

	withoutBrace := strings.TrimSpace(strings.TrimSuffix(line, "{"))
	signature := strings.TrimSpace(strings.TrimPrefix(withoutBrace, "func "))

	arrowIdx := strings.Index(signature, "->")
	if arrowIdx < 0 {
		return Function{}, fmt.Errorf("function header missing return type")
	}

	left := strings.TrimSpace(signature[:arrowIdx])
	retType := strings.TrimSpace(signature[arrowIdx+2:])
	if retType == "" {
		return Function{}, fmt.Errorf("function return type is empty")
	}

	openIdx := strings.Index(left, "(")
	closeIdx := strings.LastIndex(left, ")")
	if openIdx <= 0 || closeIdx < openIdx {
		return Function{}, fmt.Errorf("function parameter list is malformed")
	}

	name := strings.TrimSpace(left[:openIdx])
	if !strings.HasPrefix(name, "@") {
		return Function{}, fmt.Errorf("function name must start with '@': %q", name)
	}

	paramsRaw := strings.TrimSpace(left[openIdx+1 : closeIdx])
	params, err := parseParams(paramsRaw)
	if err != nil {
		return Function{}, err
	}

	return Function{Name: name, Parameters: params, ReturnType: retType}, nil
}

func parseParams(raw string) ([]Parameter, error) {
	if raw == "" {
		return nil, nil
	}

	parts := strings.Split(raw, ",")
	params := make([]Parameter, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		pieces := strings.Split(part, ":")
		if len(pieces) != 2 {
			return nil, fmt.Errorf("malformed parameter %q", part)
		}
		name := strings.TrimSpace(pieces[0])
		typ := strings.TrimSpace(pieces[1])
		if !strings.HasPrefix(name, "%") {
			return nil, fmt.Errorf("parameter name must start with '%%': %q", name)
		}
		if typ == "" {
			return nil, fmt.Errorf("parameter type is empty for %q", name)
		}
		params = append(params, Parameter{Name: name, Type: typ})
	}
	return params, nil
}

func parseInstruction(line string) (Instruction, error) {
	if strings.HasSuffix(line, ":") {
		label := strings.TrimSuffix(line, ":")
		if !strings.HasPrefix(label, ".L") {
			return Instruction{}, fmt.Errorf("invalid label %q", label)
		}
		return Instruction{Kind: InstructionLabel, Label: label}, nil
	}

	if strings.HasPrefix(line, "jmp ") {
		label := strings.TrimSpace(strings.TrimPrefix(line, "jmp "))
		if !strings.HasPrefix(label, ".L") {
			return Instruction{}, fmt.Errorf("jmp target must be a label: %q", label)
		}
		return Instruction{Kind: InstructionJmp, Label: label}, nil
	}

	if strings.HasPrefix(line, "br ") {
		rest := strings.TrimSpace(strings.TrimPrefix(line, "br "))
		parts := splitCommaSeparated(rest)
		if len(parts) != 3 {
			return Instruction{}, fmt.Errorf("br requires 3 operands")
		}
		if !strings.HasPrefix(parts[1], ".L") || !strings.HasPrefix(parts[2], ".L") {
			return Instruction{}, fmt.Errorf("br targets must be labels")
		}
		return Instruction{
			Kind:       InstructionBr,
			Condition:  parts[0],
			TrueLabel:  parts[1],
			FalseLabel: parts[2],
		}, nil
	}

	if line == "ret" || strings.HasPrefix(line, "ret ") {
		value := strings.TrimSpace(strings.TrimPrefix(line, "ret"))
		return Instruction{Kind: InstructionRet, ReturnValue: value}, nil
	}

	inst := Instruction{Kind: InstructionOp}
	right := line
	if eqIdx := strings.Index(line, "="); eqIdx >= 0 {
		left := strings.TrimSpace(line[:eqIdx])
		right = strings.TrimSpace(line[eqIdx+1:])
		if !strings.HasPrefix(left, "%") {
			return Instruction{}, fmt.Errorf("destination must start with '%%': %q", left)
		}
		inst.Destination = left
	}

	tokens := strings.Fields(right)
	if len(tokens) == 0 {
		return Instruction{}, fmt.Errorf("empty instruction")
	}
	inst.Opcode = tokens[0]
	if _, ok := coreOpcodes[inst.Opcode]; !ok {
		if _, optional := optionalOpcodes[inst.Opcode]; optional {
			return Instruction{}, fmt.Errorf("opcode %q is recognized but not enabled in milestone M1", inst.Opcode)
		}
		return Instruction{}, fmt.Errorf("unknown opcode %q", inst.Opcode)
	}

	rest := strings.TrimSpace(strings.TrimPrefix(right, inst.Opcode))
	if rest != "" {
		if inst.Opcode == "call" {
			inst.Operands = []string{rest}
		} else {
			inst.Operands = splitCommaSeparated(rest)
		}
	}
	return inst, nil
}

func splitCommaSeparated(raw string) []string {
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			values = append(values, part)
		}
	}
	return values
}

func stripComment(line string) string {
	idx := strings.Index(line, ";")
	if idx < 0 {
		return line
	}
	return line[:idx]
}

func (p *parser) errf(format string, args ...any) error {
	return fmt.Errorf("line %d: %s", p.line, fmt.Sprintf(format, args...))
}

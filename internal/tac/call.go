package tac

import (
	"fmt"
	"strings"
)

func parseCallText(raw string) (string, []ValueRef, error) {
	raw = strings.TrimSpace(raw)
	open := strings.Index(raw, "(")
	close := strings.LastIndex(raw, ")")
	if open <= 0 || close < open || close != len(raw)-1 || open != strings.LastIndex(raw, "(") || close != strings.Index(raw, ")") {
		return "", nil, fmt.Errorf("malformed call operand %q", raw)
	}
	callee := strings.TrimSpace(raw[:open])
	if !strings.HasPrefix(callee, "@") {
		return "", nil, fmt.Errorf("malformed call callee %q", callee)
	}
	argsRaw := strings.TrimSpace(raw[open+1 : close])
	if argsRaw == "" {
		return callee, nil, nil
	}
	parts := strings.Split(argsRaw, ",")
	args := make([]ValueRef, 0, len(parts))
	for _, p := range parts {
		arg := strings.TrimSpace(p)
		if arg == "" {
			return "", nil, fmt.Errorf("malformed call operand %q", raw)
		}
		args = append(args, ValueRef(arg))
	}
	return callee, args, nil
}

func formatCallText(callee string, args []ValueRef) string {
	if len(args) == 0 {
		return callee + "()"
	}
	parts := make([]string, 0, len(args))
	for _, a := range args {
		parts = append(parts, string(a))
	}
	return callee + "(" + strings.Join(parts, ", ") + ")"
}

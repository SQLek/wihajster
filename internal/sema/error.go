package sema

import (
	"fmt"

	"github.com/SQLek/wihajster/internal/lexer"
)

type Error struct {
	Line    int
	Column  int
	Message string
}

func (e *Error) Error() string {
	return fmt.Sprintf("%d:%d: %s", e.Line, e.Column, e.Message)
}

func newError(tok lexer.Token, format string, args ...any) *Error {
	return &Error{Line: tok.Line, Column: tok.Column, Message: fmt.Sprintf(format, args...)}
}

func unsupportedError(tok lexer.Token, feature string) *Error {
	return newError(tok, "unsupported in current subset: %s", feature)
}

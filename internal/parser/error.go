package parser

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
	return &Error{
		Line:    tok.Line,
		Column:  tok.Column,
		Message: fmt.Sprintf(format, args...),
	}
}

func unsupportedError(tok lexer.Token, feature string) *Error {
	return newError(tok, "unsupported in current subset: %s", feature)
}

type ParseErrors struct {
	FatalLexer  error
	Diagnostics []*Error
}

func (e *ParseErrors) Error() string {
	if e == nil {
		return ""
	}
	if e.FatalLexer != nil {
		return fmt.Sprintf("lexer error: %v", e.FatalLexer)
	}
	if len(e.Diagnostics) > 0 {
		return e.Diagnostics[0].Error()
	}
	return "parse failed"
}

func (e *ParseErrors) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.FatalLexer
}

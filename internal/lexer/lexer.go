package lexer

import (
	"fmt"
	"io"
	"sync"
)

type Lexer struct {
	mx sync.Mutex

	scanner io.ByteScanner

	line, column int
}

func New(scanner io.ByteScanner) *Lexer {
	return &Lexer{
		scanner: scanner,
		line:    1,
	}
}

func (l *Lexer) next() (Token, error) {
	switch b, err := l.scanner.ReadByte(); {

	case err != nil:
		return Token{}, err

	case b == '\n':
		l.line++
		l.column = 0
		return l.next()

	case charClassNonDigit.contains(b):
		return l.readIdentifier()

	case b == '.':
		return l.readDecimalFloatConstant("")

	case b == '0':
		return l.readOctalOrHexadecimalConstant()

	case charClassDigit.contains(b):
		// 0 case handled above, so we know it's 1-9
		l.scanner.UnreadByte()
		return l.readDecimalConstant()

	default:
		return Token{}, fmt.Errorf("unexpected character %x at line %d, column %d", b, l.line, l.column)
	}
}

func (l *Lexer) readIdentifier() (Token, error) {
	l.scanner.UnreadByte()

	value, err := charClassIdentifier.collectFrom(l.scanner)
	if err != nil {
		return Token{}, err
	}
	token := Token{
		Type:   TokenIdentifier,
		Value:  value,
		Line:   l.line,
		Column: l.column,
	}
	l.column += len(value)
	return token, nil
}

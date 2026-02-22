package lexer

import (
	"fmt"
	"io"
)

type tokenizer struct {
	scanner io.ByteScanner

	line, column int
}

func (l *tokenizer) next() (Token, error) {
	switch b, err := l.scanner.ReadByte(); {
	case err == io.EOF:
		if closer, ok := l.scanner.(io.Closer); ok {
			err = closer.Close()
		}
		return Token{Type: TokenEOF}, err

	case err != nil:
		return Token{}, err

	case charClassWhitespace.contains(b):
		// whitespace tends to be more than one character, so we collect it in one go
		return l.readWhitespace()

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

	case b == '/':
		return l.readPunctuationOrComment()

	case charClassPunctuation.contains(b):
		return l.readPunctuation(b)

	default:
		return Token{}, fmt.Errorf("unexpected character %q at line %d, column %d", b, l.line, l.column)
	}
}

func (l *tokenizer) readIdentifier() (Token, error) {
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

func (l *tokenizer) readPunctuationOrComment() (Token, error) {
	// we know that '/' is already read
	b, err := l.scanner.ReadByte()
	if err != nil {
		return Token{}, err
	}
	switch b {
	case '/':
		return l.readSingleLineComment()
	case '*':
		return l.readMultiLineComment()
	default:
		return l.readPunctuation('/')
	}
}

func (l *tokenizer) readWhitespace() (Token, error) {
	value, err := charClassWhitespace.collectFrom(l.scanner)
	if err != nil && err != io.EOF {
		return Token{}, err
	}

	// whitespace can have multiple lines, so we need to count them to update line and column correctly
	lines := 0
	lastNewLineIndex := -1

	for i, c := range value {
		if c == '\n' {
			lines++
			lastNewLineIndex = i
		}
	}

	l.line += lines
	l.column = len(value) - lastNewLineIndex - 1

	return l.next()
}

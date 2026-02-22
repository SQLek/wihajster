package lexer

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

func (l *tokenizer) readStringLiteral(isLong bool) (Token, error) {
	// we had read " pr L"
	var builder strings.Builder
	if isLong {
		builder.WriteByte('L')
	}
	builder.WriteByte('"')

	// we can have line continuation that could be hard to count
	// we should store token start
	startLine := l.line
	startColumn := l.column

	for {
		v, err := charClassStringLiteralEnd.collectNegatingFrom(l.scanner)
		if err != nil {
			return Token{}, err
		}
		builder.WriteString(v)
		l.column += len(v)

		// we expect escape sequence or end of character constant
		v, isEnd, err := l.readEscapeSequenceOrEnd()
		if err != nil {
			return Token{}, err
		}
		if isEnd {
			break
		}
		builder.WriteString(v)
	}
	builder.WriteByte('"') // closing quote
	l.column++             // closing quote

	return Token{
		Type:   TokenStringLiteral,
		Value:  builder.String(),
		Line:   startLine,
		Column: startColumn,
	}, nil
}

func (l *tokenizer) readCharacterConstant(isLong bool) (Token, error) {
	// we had read ' pr L'
	var builder strings.Builder
	if isLong {
		builder.WriteByte('L')
	}
	builder.WriteByte('\'')

	// we can have line continuation that could be hard to count
	// we should store token start
	startLine := l.line
	startColumn := l.column

	for {
		v, err := charClassCharacterConstantEnd.collectNegatingFrom(l.scanner)
		if err != nil {
			return Token{}, err
		}
		builder.WriteString(v)
		l.column += len(v)

		// we expect escape sequence or end of character constant
		v, isEnd, err := l.readEscapeSequenceOrEnd()
		if err != nil {
			return Token{}, err
		}
		if isEnd {
			break
		}
		builder.WriteString(v)
	}
	builder.WriteByte('\'') // closing quote
	l.column++              // closing quote

	return Token{
		Type:   TokenCharacterConstant,
		Value:  builder.String(),
		Line:   startLine,
		Column: startColumn,
	}, nil
}

func (l *tokenizer) readEscapeSequenceOrEnd() (string, bool, error) {
	var buff []byte
	err := expectOneOfAndAppend(l.scanner, `'"\\`, &buff)
	switch {
	case err != nil:
		return "", false, err
	case len(buff) == 0:
		return "", false, fmt.Errorf("unexpected EOF")
	case buff[0] == '\'' || buff[0] == '"':
		l.column++
		return string(buff), true, nil
	}

	b, err := l.scanner.ReadByte()
	if err == io.EOF {
		return "", false, fmt.Errorf("unexpected EOF in escape sequence")
	}
	if err != nil {
		return "", false, err
	}

	switch {
	// special handling escapes
	case b == '\n':
		l.line++
		l.column = 1
		return "", false, nil
	case b == 'x':
		v, err := l.readHexadecimalEscapeSequence()
		return v, false, err
	case charClassOctalDigit.contains(b):
		v, err := l.readOctalEscapeSequence(b)
		return v, false, err
	}

	switch b {
	case '\n':
		// do nothing, inform about line continuation
		return "", true, nil
	case '\'':
		return "'", false, nil
	case '"':
		return "\"", false, nil
	case '?':
		return "?", false, nil
	case '\\':
		return "\\", false, nil
	case 'a':
		return "\a", false, nil
	case 'b':
		return "\b", false, nil
	case 'f':
		return "\f", false, nil
	case 'n':
		return "\n", false, nil
	case 'r':
		return "\r", false, nil
	case 't':
		return "\t", false, nil
	case 'v':
		return "\v", false, nil

	case 'x':
		// hexadecimal escape sequence, we need at least one hex digit
		v, err := charClassHexadecimalDigit.collectFrom(l.scanner)
		if err != nil {
			return "", false, fmt.Errorf("invalid hexadecimal escape sequence: %w", err)
		}
		return string(v), false, nil

	default:
		return "", false, fmt.Errorf("invalid escape sequence: \\%c", b)
	}
}

func (l *tokenizer) readOctalEscapeSequence(first byte) (string, error) {
	buff := []byte{first}
	for i := 0; i < 2; i++ {
		err := charClassOctalDigit.collectOneOfFromAndAppend(l.scanner, &buff)
		if err != nil {
			return "", err
		}
		if len(buff) == 1+i {
			// we didn't read another octal digit, so we can stop
			break
		}
	}

	l.column += len(buff) + 1 // \ + octal digits

	v, _ := strconv.ParseInt(string(buff), 8, 8) // we can ignore error, we know it's valid octal
	return string(rune(v)), nil
}

func (l *tokenizer) readHexadecimalEscapeSequence() (string, error) {
	v, err := charClassHexadecimalDigit.collectFrom(l.scanner)
	if err != nil {
		return "", fmt.Errorf("invalid hexadecimal escape sequence: %w", err)
	}
	if len(v) == 0 {
		return "", fmt.Errorf("invalid hexadecimal escape sequence: at least one hexadecimal digit is required")
	}

	l.column += len(v) + 2                        // \x + hexadecimal digits
	vInt, _ := strconv.ParseInt(string(v), 16, 8) // we can ignore error, we know it's valid hexadecimal
	return string(rune(vInt)), nil
}

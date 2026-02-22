package lexer

import (
	"cmp"
	"io"
)

func (l *Lexer) readDecimalConstant() (Token, error) {
	value, err := charClassDigit.collectFrom(l.scanner)
	if err != nil {
		return Token{}, err
	}

	// handle float part that could have . or eE exponent
	b, err := l.scanner.ReadByte()
	if err != nil && err != io.EOF {
		return Token{}, err
	}
	if b == '.' {
		return l.readDecimalFloatConstant(value)
	}
	if err == nil {
		// if there EOF, we cannot unread dot.
		l.scanner.UnreadByte()
	}

	ok, dToken, err := l.tryReadDecimalNonFractionalFloatConstant(value)
	if err != nil {
		return Token{}, err
	}
	if ok {
		return dToken, nil
	}

	suffix, err := readLLUSuffix(l.scanner)
	if err != nil {
		return Token{}, err
	}

	token := Token{
		Type:   TokenIntegerConstant,
		Value:  value + suffix,
		Line:   l.line,
		Column: l.column,
	}
	l.column += len(value + suffix)
	return token, nil
}

func (l *Lexer) readOctalOrHexadecimalConstant() (Token, error) {
	// 0 octal-digits or 0x hexadecimal-digits
	b, err := l.scanner.ReadByte()
	if err != nil {
		return Token{}, err
	}

	if b == 'x' || b == 'X' {
		return l.readHexadecimalConstant(b)
	}

	l.scanner.UnreadByte()
	return l.readOctalConstant()
}

func (l *Lexer) readOctalConstant() (Token, error) {
	value, err := charClassOctalDigit.collectFrom(l.scanner)
	if err != nil {
		return Token{}, err
	}

	// octals don't have float variants, so we can read suffixes right away

	suffix, err := readLLUSuffix(l.scanner)
	if err != nil {
		return Token{}, err
	}

	token := Token{
		Type:   TokenIntegerConstant,
		Value:  "0" + value + suffix,
		Line:   l.line,
		Column: l.column,
	}
	l.column += len("0" + value + suffix)
	return token, nil
}

func (l *Lexer) readHexadecimalConstant(xChar byte) (Token, error) {
	value, err := charClassHexadecimalDigit.collectFrom(l.scanner)
	if err != nil {
		return Token{}, err
	}

	// heaxedecimals can have fraction with . here, or binary exponent with pP
	b, err := l.scanner.ReadByte()
	if err != nil && err != io.EOF {
		return Token{}, err
	}
	if b == '.' {
		return l.readHexadecimalFloatConstant(xChar, value)
	}
	if err == nil {
		// if there EOF, we cannot unread dot.
		l.scanner.UnreadByte()
	}

	ok, fToken, err := l.tryReadHexadecimalNonFractionalFloatConstant(xChar, value)
	if err != nil {
		return Token{}, err
	}
	if ok {
		return fToken, nil
	}

	suffix, err := readLLUSuffix(l.scanner)
	if err != nil {
		return Token{}, err
	}

	token := Token{
		Type:   TokenIntegerConstant,
		Value:  "0" + string(xChar) + value + suffix,
		Line:   l.line,
		Column: l.column,
	}
	l.column += len("0" + string(xChar) + value + suffix)
	return token, nil
}

func readLLUSuffix(scanner io.ByteScanner) (string, error) {
	var buff []byte
	err1 := expectOneOfAndAppend(scanner, "lL", &buff)
	err2 := expectOneOfAndAppend(scanner, "lL", &buff)
	err3 := expectOneOfAndAppend(scanner, "uU", &buff)

	if err := cmp.Or(err1, err2, err3); err != nil && err != io.EOF {
		return "", err
	}

	return string(buff), nil
}

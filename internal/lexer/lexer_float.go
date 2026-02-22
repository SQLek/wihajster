package lexer

import (
	"cmp"
	"io"
)

func (l *Lexer) readDecimalFloatConstant(digits string) (Token, error) {
	fractional, err := charClassDigit.collectFrom(l.scanner)
	if err != nil {
		return Token{}, err
	}

	fractional = digits + "." + fractional
	exponent, err := l.readFloatExponent(charClassDecimalExponent)
	if err != nil {
		return Token{}, err
	}

	suffix, err := readFLSuffix(l.scanner)
	if err != nil {
		return Token{}, err
	}

	token := Token{
		Type:   TokenFloatingConstant,
		Value:  fractional + exponent + suffix,
		Line:   l.line,
		Column: l.column,
	}
	l.column += len(fractional + exponent + suffix)
	return token, nil
}

func (l *Lexer) tryReadDecimalNonFractionalFloatConstant(digits string) (bool, Token, error) {
	exponent, err := l.readFloatExponent(charClassDecimalExponent)
	if err != nil {
		return false, Token{}, err
	}
	if exponent == "" {
		return false, Token{}, nil
	}

	suffix, err := readFLSuffix(l.scanner)
	if err != nil {
		return false, Token{}, err
	}

	token := Token{
		Type:   TokenFloatingConstant,
		Value:  digits + exponent + suffix,
		Line:   l.line,
		Column: l.column,
	}
	l.column += len(digits + exponent + suffix)
	return true, token, nil
}

func (l *Lexer) readHexadecimalFloatConstant(xChar byte, digits string) (Token, error) {
	fractional, err := charClassHexadecimalDigit.collectFrom(l.scanner)
	if err != nil {
		return Token{}, err
	}

	fractional = digits + "." + fractional
	exponent, err := l.readFloatExponent(charClassHexadecimalExponent)
	if err != nil {
		return Token{}, err
	}

	suffix, err := readFLSuffix(l.scanner)
	if err != nil {
		return Token{}, err
	}

	token := Token{
		Type:   TokenFloatingConstant,
		Value:  "0" + string(xChar) + fractional + exponent + suffix,
		Line:   l.line,
		Column: l.column,
	}
	l.column += len("0" + string(xChar) + fractional + exponent + suffix)
	return token, nil
}

func (l *Lexer) tryReadHexadecimalNonFractionalFloatConstant(xChar byte, digits string) (bool, Token, error) {
	exponent, err := l.readFloatExponent(charClassHexadecimalExponent)
	if err != nil {
		return false, Token{}, err
	}
	if exponent == "" {
		return false, Token{}, nil
	}

	suffix, err := readFLSuffix(l.scanner)
	if err != nil {
		return false, Token{}, err
	}

	token := Token{
		Type:   TokenFloatingConstant,
		Value:  "0" + string(xChar) + digits + exponent + suffix,
		Line:   l.line,
		Column: l.column,
	}
	l.column += len("0" + string(xChar) + digits + exponent + suffix)
	return true, token, nil
}

// Reads the exponent part of a floating constant, both decimal and hexadecimal have decimal exponent, but eE or pP prefix respectively
func (l *Lexer) readFloatExponent(exponent charClass) (string, error) {
	var buff []byte

	e, err := l.scanner.ReadByte()
	if err == io.EOF {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if !exponent.contains(e) {
		l.scanner.UnreadByte()
		return "", nil
	}
	buff = append(buff, e)

	err = expectOneOfAndAppend(l.scanner, "+-", &buff)
	if err != nil && err != io.EOF {
		return "", err
	}

	value, err := charClassDigit.collectFrom(l.scanner)
	if err != nil {
		return "", err
	}

	return string(buff) + value, nil
}

func readFLSuffix(scanner io.ByteScanner) (string, error) {
	var buff []byte
	err1 := expectOneOfAndAppend(scanner, "fF", &buff)
	err2 := expectOneOfAndAppend(scanner, "lL", &buff)

	if err := cmp.Or(err1, err2); err != nil && err != io.EOF {
		return "", err
	}

	return string(buff), nil
}

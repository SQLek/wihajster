package lexer

import (
	"io"
	"strings"
)

type charClass string

const (
	charClassNonDigit charClass = "_abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

	charClassDigit charClass = "0123456789"

	charClassOctalDigit charClass = "01234567"

	charClassHexadecimalDigit charClass = charClassDigit + "abcdefABCDEF"

	charClassIdentifier = charClassNonDigit + charClassDigit

	charClassDecimalExponent charClass = "eE"

	charClassHexadecimalExponent charClass = "pP"
)

func (cc charClass) contains(b byte) bool {
	return strings.ContainsRune(string(cc), rune(b))
}

func (cc charClass) collectFrom(scanner io.ByteScanner) (string, error) {
	var buff []byte
	for {
		b, err := scanner.ReadByte()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if !cc.contains(b) {
			scanner.UnreadByte()
			break
		}
		buff = append(buff, b)
	}
	return string(buff), nil
}

func expectOneOfAndAppend(scanner io.ByteScanner, chars string, buff *[]byte) error {
	b, err := scanner.ReadByte()
	if err != nil {
		return err
	}
	for i := 0; i < len(chars); i++ {
		// C language is byte and not utf8 oriented, byte iteration over chars is fine
		if chars[i] == b {
			*buff = append(*buff, b)
			return nil
		}
	}
	scanner.UnreadByte()
	return nil
}

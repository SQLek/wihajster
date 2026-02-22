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

	charClassNewLine charClass = "\n"

	charClsassSinglePunctuation charClass = "[](){}~?;,"

	charClassMultiPunctuation charClass = ".-+*/%&|^!=<>:#"

	charClassPunctuation = charClsassSinglePunctuation + charClassMultiPunctuation

	charClassWhitespace charClass = " \t\r\n\f"

	charClassCharacterConstantEnd charClass = `'\n\`

	charClassStringLiteralEnd charClass = `"\n\`

	charClassSimpleEscapeSequence charClass = `\'"?\\abfnrtv`
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

func (cc charClass) collectNegatingFrom(scanner io.ByteScanner) (string, error) {
	var buff []byte
	for {
		b, err := scanner.ReadByte()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if cc.contains(b) {
			scanner.UnreadByte()
			break
		}
		buff = append(buff, b)
	}
	return string(buff), nil
}

func (cc charClass) collectOneOfFromAndAppend(scanner io.ByteScanner, buff *[]byte) error {
	return expectOneOfAndAppend(scanner, string(cc), buff)
}

func expectOneOfAndAppend(scanner io.ByteScanner, chars string, buff *[]byte) error {
	b, err := scanner.ReadByte()
	if err == io.EOF {
		return nil
	}
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

func expectOneOfAndBuilder(scanner io.ByteScanner, chars string, builder *strings.Builder) (bool, error) {
	b, err := scanner.ReadByte()
	if err == io.EOF {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	for i := 0; i < len(chars); i++ {
		// C language is byte and not utf8 oriented, byte iteration over chars is fine
		if chars[i] == b {
			builder.WriteByte(b)
			return true, nil
		}
	}
	scanner.UnreadByte()
	return false, nil
}

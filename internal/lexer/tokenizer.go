package lexer

import "io"

var (
	alphaByteClass = byteClassCombine(
		byteClassRange('a', 'z'),
		byteClassRange('A', 'Z'),
		byteClassChars('_'),
	)

	alphaDigitByteClass = byteClassCombine(
		alphaByteClass,
		digitByteClass,
	)
)

func lex(s *scanner, buildFn tokenBuildFn) (TokenType, error) {
	b, err := s.peekOne()
	if err != nil {
		return tokenNil, err
	}

	switch {
	case whiteByteClass.contains(b):
		return lexWhiteSpace(s, buildFn)

	case alphaByteClass.contains(b):
		return lexIdentifier(s, buildFn)

	case b == '0':
		return lexOctalOrHexadecimalConstant(s, buildFn)

	case digitByteClass.contains(b):
		return lexDecimalInteger(s, buildFn)

	case b == '\'':
		buff1Char[0] = s.popOneFromBuffer()
		buildFn(buff1Char)
		return lexCharacterConstant(s, buildFn)

	case b == '"':
		buff1Char[0] = s.popOneFromBuffer()
		buildFn(buff1Char)
		return lexStringLiteral(s, buildFn)

	case b == '.':
		return lexDots(s, buildFn)

	default:
		// lexPunctuation is robust enough to push all other into
		return lexPunctuation(s, buildFn)
	}
}

func lexIdentifier(s *scanner, buildFn tokenBuildFn) (TokenType, error) {
	data, isPartial, err := s.readBytesInClass(alphaDigitByteClass)
	if err != nil && err != io.EOF {
		return tokenNil, err
	}
	buildFn(data)
	if !isPartial {
		return TokenIdentifier, nil
	}
	return lexIdentifier(s, buildFn)
}

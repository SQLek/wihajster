package lexer

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
		return lexStringConstant(s, charConstantBody, buildFn)

	case b == '"':
		return lexStringConstant(s, stringLiteralBody, buildFn)

	case b == '.':
		return lexDots(s, buildFn)

	default:
		// lexPunctuation is robust enough to push all other into
		return lexPunctuation(s, buildFn)
	}
}

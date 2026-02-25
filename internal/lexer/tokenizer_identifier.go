package lexer

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

func lexIdentifier(s *scanner, buildFn tokenBuildFn) (TokenType, error) {
	data, isPartial, err := s.readBytesInClass(alphaDigitByteClass)
	if err != nil {
		return tokenNil, err
	}
	buildFn(data)
	if !isPartial {
		return TokenIdentifier, nil
	}
	return lexIdentifier(s, buildFn)
}

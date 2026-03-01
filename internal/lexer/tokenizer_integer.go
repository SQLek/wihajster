package lexer

import "io"

var (
	digitByteClass      = byteClassRange('0', '9')
	octalDigitByteClass = byteClassRange('0', '7')
	hexalDigitByteClass = byteClassCombine(
		digitByteClass,
		byteClassRange('a', 'f'),
		byteClassRange('A', 'F'),
	)
)

func lexDecimalInteger(s *scanner, buildFn tokenBuildFn) (TokenType, error) {
	data, isPartial, err := s.readBytesInClass(digitByteClass)
	if err != nil && err != io.EOF {
		return tokenNil, err
	}
	buildFn(data)
	if isPartial {
		return lexDecimalInteger(s, buildFn)
	}

	b, err := s.peekOne()
	if err != nil && err != io.EOF {
		return tokenNil, err
	}
	if b == '.' || b == 'e' || b == 'E' {
		return lexDecimalFloat(s, buildFn)
	}

	err = consumeLLUSuffix(s, buildFn)
	if err != nil && err != io.EOF {
		return tokenNil, err
	}

	return TokenIntegerConstant, nil
}

func lexOctalOrHexadecimalConstant(s *scanner, buildFn tokenBuildFn) (TokenType, error) {
	// 0 already peeked
	s.popOneFromBuffer()
	buff1Char[0] = '0'
	buildFn(buff1Char)

	// 0 is just 0, x denotes hexedecimal, any other octal means octal
	b, err := s.peekOne()
	if err == io.EOF {
		// 0 is till a valid octal
		return TokenIntegerConstant, nil
	}
	if err != nil {
		return tokenNil, err
	}
	if b == 'x' || b == 'X' {
		return lexHexadecimalConstant(s, buildFn)
	}

	return lexOctalConstant(s, buildFn)
}

func lexOctalConstant(s *scanner, buildFn tokenBuildFn) (TokenType, error) {
	data, isPartial, err := s.readBytesInClass(octalDigitByteClass)
	if err != nil && err != io.EOF {
		return tokenNil, err
	}
	buildFn(data)
	if isPartial {
		return lexOctalConstant(s, buildFn)
	}

	// octals don't have float variants

	err = consumeLLUSuffix(s, buildFn)
	if err != nil && err != io.EOF {
		return tokenNil, err
	}

	return TokenIntegerConstant, nil
}

func lexHexadecimalConstant(s *scanner, buildFn tokenBuildFn) (TokenType, error) {
	data, isPartial, err := s.readBytesInClass(hexalDigitByteClass)
	if err != nil && err != io.EOF {
		return tokenNil, err
	}
	buildFn(data)
	if isPartial {
		return lexHexadecimalConstant(s, buildFn)
	}

	b, err := s.peekOne()
	if err != nil {
		return tokenNil, err
	}
	if b == '.' || b == 'p' || b == 'P' {
		return lexHexedecimalFloat(s, buildFn)
	}

	err = consumeLLUSuffix(s, buildFn)
	if err != nil && err != io.EOF {
		return tokenNil, err
	}

	return TokenIntegerConstant, nil
}

// this function is to consume l, ll, llu, LlU ...
// and other cases of LLU integer suffix
// not he cleanest of code, but it get's job done.
func consumeLLUSuffix(s *scanner, buildFn tokenBuildFn) error {
	sendCurrent := func() {
		// using buff1Char because lexing not multithreaded
		buff1Char[0] = s.popOneFromBuffer()
		buildFn(buff1Char)
	}

	// first 'l/L' or 'u/U'
	b, err := s.peekOne()
	if err != nil {
		return err
	}
	if b != 'u' && b != 'U' && b != 'l' && b != 'L' {
		// something else peeded
		return nil
	}
	sendCurrent()
	if b == 'u' || b == 'U' {
		// 'u' always last
		return nil
	}

	// second 'l/L' or 'u/U'
	b, err = s.peekOne()
	if err != nil {
		return err
	}
	if b != 'u' && b != 'U' && b != 'l' && b != 'L' {
		// something else peeded
		return nil
	}
	sendCurrent()
	if b == 'u' || b == 'U' {
		// 'u' always last
		return nil
	}

	// now ne have 'll' consumed, we can still consume 'u/U
	b, err = s.peekOne()
	if err != nil {
		return err
	}
	if b != 'u' && b != 'U' {
		// something else peeded
		return nil
	}

	sendCurrent()
	return nil
}

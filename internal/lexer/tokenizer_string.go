package lexer

import "fmt"

// This file handles bot string literals and character constants
var (
	// \n can be inside string or char only in escape or line continuation
	// otherwise we rise error right away
	stringLiteralBody = byteClassChars('\n', '\\', '"').negate()
	charConstantBody  = byteClassChars('\n', '\\', '\'').negate()
)

func lexStringLiteral(s *scanner, buildFn tokenBuildFn) (TokenType, error) {
	// because of recurrent calling, both on partial but on escape sequences,
	// we have to require that start of sling literal is already read
	data, isPartial, err := s.readBytesInClass(stringLiteralBody)
	if err != nil {
		return tokenNil, err
	}
	buildFn(data)
	if isPartial {
		return lexStringLiteral(s, buildFn)
	}

	switch b, err := s.peekOne(); {
	case err != nil:
		return tokenNil, err
	case b == '"':
		buff1Char[0] = s.popOneFromBuffer()
		buildFn(buff1Char)
		return TokenStringLiteral, nil
	case b == '\n':
		return tokenNil, fmt.Errorf("unexpected newline in string literal at %d:%d", s.line, s.column)
	}

	// we have '\' at cursor, lets pop it and see if we have escape, or line continuation
	s.popOneFromBuffer()
	switch b, err := s.peekOne(); {
	case err != nil:
		return tokenNil, err

	case b == '\n':
		// line continuation \\\n
		s.popOneFromBuffer()
		return lexStringLiteral(s, buildFn)

	case b == '"':
		buff1Char[0] = s.popOneFromBuffer()
		buildFn(buff1Char)
		return lexStringLiteral(s, buildFn)

	default:
		// other escape sequencess will be handled in future
		return tokenNil, ErrNotImplementedInV0
	}
}

func lexCharacterConstant(s *scanner, buildFn tokenBuildFn) (TokenType, error) {
	// because of recurrent calling, both on partial but on escape sequences,
	// we have to require that start of sling literal is already read
	data, isPartial, err := s.readBytesInClass(charConstantBody)
	if err != nil {
		return tokenNil, err
	}
	buildFn(data)
	if isPartial {
		return lexCharacterConstant(s, buildFn)
	}

	switch b, err := s.peekOne(); {
	case err != nil:
		return tokenNil, err
	case b == '\'':
		buff1Char[0] = s.popOneFromBuffer()
		buildFn(buff1Char)
		return TokenCharacterConstant, nil
	case b == '\n':
		return tokenNil, fmt.Errorf("unexpected newline in character literal at %d:%d", s.line, s.column)
	}

	// we have '\' at cursor, lets pop it and see if we have escape, or line continuation
	s.popOneFromBuffer()
	switch b, err := s.peekOne(); {
	case err != nil:
		return tokenNil, err

	case b == '\n':
		// line continuation \\\n
		s.popOneFromBuffer()
		return lexCharacterConstant(s, buildFn)

	case b == '\'':
		buff1Char[0] = s.popOneFromBuffer()
		buildFn(buff1Char)
		return lexCharacterConstant(s, buildFn)

	default:
		// other escape sequencess will be handled in future
		return tokenNil, ErrNotImplementedInV0
	}
}

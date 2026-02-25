package lexer

var whiteByteClass = byteClassChars(' ', '\n', '\r', '\t')

// consumes all whitespace
func lexWhiteSpace(s *scanner, buildFn tokenBuildFn) (TokenType, error) {
	// no matter if is complete or not, we run lex anyway
	_, _, err := s.readBytesInClass(whiteByteClass)
	if err != nil {
		return tokenNil, err
	}

	return lex(s, buildFn)
}

// single line comment can at \n with exception of line continuation '\' '\n'
var bcSingleLineCommentEnd = byteClassChars('\n', '\\').negate()

func lexCommentSingleLine(s *scanner, buildFn tokenBuildFn) (TokenType, error) {
	_, isPartial, err := s.readBytesInClass(bcSingleLineCommentEnd)
	if err != nil {
		return tokenNil, err
	}
	if isPartial {
		return lexCommentSingleLine(s, buildFn)
	}

	// now we have line continuation or comment end - consuming anyway
	b, err := s.readOne()
	if err != nil {
		return tokenNil, err
	}
	if b == '\\' {
		// line continuation, caling self recursively
		return lexCommentSingleLine(s, buildFn)
	}

	return lex(s, buildFn)
}

func lexCommentMultiLine(s *scanner, buildFn tokenBuildFn) (TokenType, error) {

	return tokenNil, ErrNotImplementedInV0
}

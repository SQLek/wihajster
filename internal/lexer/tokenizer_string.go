package lexer

// This file handles bot string literals and character constants
var (
	// \n can be inside string or char only in escape or line continuation
	// otherwise we rise error right away
	stringLiteralBody = byteClassChars('\n', '\\', '"')
	charConstantBody  = byteClassChars('\n', '\\', '\'')
)

func lexStringConstant(s *scanner, bodyBC byteClass, buildFn tokenBuildFn) (TokenType, error) {

	return tokenNil, ErrNotImplementedInV0
}

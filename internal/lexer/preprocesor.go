package lexer

// from preprocesor point of view many characters are irrleevant except...
var preProcIrrelevantByteClass = byteClassCombine(
	// start of comment, string literal, character constant and ofc '#'
	byteClassChars('/', '\'', '"', '#'),

// but as a preprocessor, our role is to substitude macros
// so we should not ignore ident start
).negate()

type preprocesor struct {
	s *scanner
}

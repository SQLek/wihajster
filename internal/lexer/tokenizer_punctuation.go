package lexer

import "fmt"

// Dots and elypsis. We have 1 character look ahead. We cannot distinguish,
// and we cannot emit two tokens at once. Must handle in preprocesor.
// Other nasty one is a `%:%:`. Full handling in preprocesor.
//
// Three char tokens we can handle, because without equal at end,
// it is still valid token. Handling like 2 char and having if for '<<=' and '>>='
var (
	// many two char punctuations are just single char and equal
	punctuationCharEqualLexTable = [256]TokenType{
		'-': tokenPunctuationTBD,
		'+': tokenPunctuationTBD,
		'&': tokenPunctuationTBD,
		'*': tokenPunctuationTBD,
		'!': tokenPunctuationTBD,
		'/': tokenPunctuationTBD,
		'%': tokenPunctuationTBD,
		'<': tokenPunctuationTBD,
		'>': tokenPunctuationTBD,
		'^': tokenPunctuationTBD,
		'|': tokenPunctuationTBD,
		'=': tokenPunctuationTBD,
	}
	punctuation2CharLexTable = [256][256]TokenType{
		'#': {'#': tokenPreProcGlue},
		'-': {'>': tokenPunctuationTBD, '-': tokenPunctuationTBD},
		'+': {'+': tokenPunctuationTBD},
		'&': {'&': tokenPunctuationTBD},
		'/': {'/': tokenCommentSingle, '*': tokenCommentMulti},
		'%': {'>': tokenPunctuationTBD, ':': tokenPunctuationTBD},
		'<': {'<': tokenShiftLeft, ':': tokenPunctuationTBD, '%': tokenPunctuationTBD},
		'>': {'>': tokenShiftRight},
		':': {'>': tokenPunctuationTBD},
		'|': {'|': tokenPunctuationTBD},
	}
	punctuation1CharLexTable = [256]TokenType{
		'#': tokenPreprocStart,
		'-': tokenPunctuationTBD,
		'+': tokenPunctuationTBD,
		'&': tokenPunctuationTBD,
		'*': tokenPunctuationTBD,
		'!': tokenPunctuationTBD,
		'/': tokenPunctuationTBD,
		'%': tokenPunctuationTBD,
		'<': tokenPunctuationTBD,
		'>': tokenPunctuationTBD,
		'^': tokenPunctuationTBD,
		':': tokenPunctuationTBD,
		'|': tokenPunctuationTBD,
		'=': tokenPunctuationTBD,
		'[': tokenPunctuationTBD,
		']': tokenPunctuationTBD,
		'(': tokenPunctuationTBD,
		')': tokenPunctuationTBD,
		'{': tokenPunctuationTBD,
		'}': tokenPunctuationTBD,
		'~': tokenPunctuationTBD,
		'?': tokenPunctuationTBD,
		';': tokenPunctuationTBD,
		',': tokenPunctuationTBD,
	}
	// reusable 1 and 2 character buffers
	buff1Char = []byte{' '}
	buff2Char = []byte{' ', ' '}
)

func lexPunctuation(s *scanner, buildFn tokenBuildFn) (TokenType, error) {
	first := s.popOneFromBuffer()
	second, err := s.peekOne()
	if err != nil {
		return tokenNil, err
	}

	sendTwo := func() {
		s.popOneFromBuffer()
		buff2Char[0] = first
		buff2Char[1] = second
		buildFn(buff2Char)
	}

	// checking two char punctuations with eqal as second char
	if tt := punctuationCharEqualLexTable[first]; second == '=' && tt != tokenNil {
		sendTwo()
		return tt, nil
	}

	// checking other two chars
	// with special case for '//', '/*', '<<=', '>>='
	switch tt := punctuation2CharLexTable[first][second]; tt {
	case tokenNil:
		// not a valid 2 char punctuation - we try single char bellow

	default:
		// not special case
		sendTwo()
		return tt, nil

	case tokenCommentSingle:
		s.popOneFromBuffer()
		return lexCommentSingleLine(s, buildFn)

	case tokenCommentMulti:
		s.popOneFromBuffer()
		return lexCommentMultiLine(s, buildFn)

	case tokenShiftLeft, tokenShiftRight:
		sendTwo()
		third, err := s.peekOne()
		if err != nil {
			return tokenNil, err
		}
		if third != '=' {
			return tt, nil
		}

		buff1Char[0] = s.popOneFromBuffer()
		if tt == tokenShiftLeft {
			return tokenPunctuationTBD, nil
		} else {
			return tokenPunctuationTBD, nil
		}
	}

	// and anything thats left is a single char punctuation
	if tt := punctuation1CharLexTable[first]; tt != tokenNil {
		return tt, nil
	}

	return tokenNil, fmt.Errorf(
		"invalid character in punctuation '%c%c' %w",
		first, second, ErrNotImplementedInV0,
	)
}

var dotsByteClass = byteClassChars('.')

func lexDots(s *scanner, buildFn tokenBuildFn) (TokenType, error) {
	data, isPartial, err := s.readBytesInClass(dotsByteClass)
	if err != nil {
		return tokenNil, err
	}
	buildFn(data)
	if !isPartial {
		return tokenDots, nil
	}
	return lexDots(s, buildFn)
}

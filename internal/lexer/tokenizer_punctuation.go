package lexer

import (
	"fmt"
	"io"
)

// Dots and elypsis. We have 1 character look ahead. We cannot distinguish,
// and we cannot emit two tokens at once. Must handle in preprocesor.
// Other nasty one is a `%:%:`. Full handling in preprocesor.
//
// Three char tokens we can handle, because without equal at end,
// it is still valid token. Handling like 2 char and having if for '<<=' and '>>='
var (
	// many two char punctuations are just single char and equal
	punctuationCharEqualLexTable = [256]TokenType{
		'-': TokenMinusAssign,
		'+': TokenPlusAssign,
		'&': TokenAmpAssign,
		'*': TokenStarAssign,
		'!': TokenNe,
		'/': TokenSlashAssign,
		'%': TokenPercentAssign,
		'<': TokenLe,
		'>': TokenGe,
		'^': TokenCaretAssign,
		'|': TokenPipeAssign,
		'=': TokenEq,
	}
	punctuation2CharLexTable = [256][256]TokenType{
		'#': {'#': tokenPreProcGlue},
		'-': {'>': TokenArrow, '-': TokenMinusMinus},
		'+': {'+': TokenPlusPlus},
		'&': {'&': TokenAndAnd},
		'/': {'/': tokenCommentSingle, '*': tokenCommentMulti},
		'%': {'>': TokenRBrace, ':': tokenPreprocStart},
		'<': {'<': TokenShiftLeft, ':': TokenLBracket, '%': TokenLBrace},
		'>': {'>': TokenShiftRight},
		':': {'>': TokenRBracket},
		'|': {'|': TokenOrOr},
	}
	punctuation1CharLexTable = [256]TokenType{
		'#': tokenPreprocStart,
		'-': TokenMinus,
		'+': TokenPlus,
		'&': TokenAmp,
		'*': TokenStar,
		'!': TokenBang,
		'/': TokenSlash,
		'%': TokenPercent,
		'<': TokenLt,
		'>': TokenGt,
		'^': TokenCaret,
		':': TokenColon,
		'|': TokenPipe,
		'=': TokenAssign,
		'[': TokenLBracket,
		']': TokenRBracket,
		'(': TokenLParen,
		')': TokenRParen,
		'{': TokenLBrace,
		'}': TokenRBrace,
		'~': TokenTilde,
		'?': TokenQuestion,
		';': TokenSemicolon,
		',': TokenComma,
		'.': TokenDot,
	}
	// reusable 1 and 2 character buffers
	buff1Char = []byte{' '}
	buff2Char = []byte{' ', ' '}
)

func lexPunctuation(s *scanner, buildFn tokenBuildFn) (TokenType, error) {
	first := s.popOneFromBuffer()
	second, err := s.peekOne()
	if err != nil && err != io.EOF {
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

	case TokenShiftLeft, TokenShiftRight:
		sendTwo()
		third, err := s.peekOne()
		if err != nil && err != io.EOF {
			return tokenNil, err
		}
		if third != '=' {
			return tt, nil
		}

		buff1Char[0] = s.popOneFromBuffer()
		buildFn(buff1Char)
		if tt == TokenShiftLeft {
			return TokenShiftLeftAssign, nil
		} else {
			return TokenShiftRightAssign, nil
		}
	}

	// and anything thats left is a single char punctuation
	if tt := punctuation1CharLexTable[first]; tt != tokenNil {
		buff1Char[0] = first
		buildFn(buff1Char)
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
	if err != nil && err != io.EOF {
		return tokenNil, err
	}
	buildFn(data)
	if !isPartial {
		return tokenDots, nil
	}
	return lexDots(s, buildFn)
}

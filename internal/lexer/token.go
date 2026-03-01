package lexer

type TokenType int

const (
	// special private token to symbolize lookahead contains empty
	tokenNil TokenType = iota

	// Keywords
	TokenAuto
	TokenEnum
	TokenRestrict
	TokenUnsigned
	TokenBreak
	TokenExtern
	TokenReturn
	TokenVoid
	TokenCase
	TokenFloat
	TokenShort
	TokenVolatile
	TokenChar
	TokenFor
	TokenSigned
	TokenWhile
	TokenConst
	TokenGoto
	TokenSizeof
	Token_Bool
	TokenContinue
	TokenIf
	TokenStatic
	Token_Complex
	TokenDefault
	TokenInline
	TokenStruct
	Token_Imaginary
	TokenDo
	TokenInt
	TokenSwitch
	TokenDouble
	TokenLong
	TokenTypedef
	TokenElse
	TokenRegister
	TokenUnion

	TokenIdentifier
	TokenIntegerConstant
	TokenFloatingConstant
	TokenCharacterConstant
	TokenStringLiteral

	// Punctuation
	TokenLBracket
	TokenRBracket
	TokenLParen
	TokenRParen
	TokenLBrace
	TokenRBrace
	TokenDot
	TokenEllipsis
	TokenArrow
	TokenPlusPlus
	TokenMinusMinus
	TokenAmp
	TokenStar
	TokenPlus
	TokenMinus
	TokenTilde
	TokenBang
	TokenSlash
	TokenPercent
	TokenShiftLeft
	TokenShiftRight
	TokenLt
	TokenGt
	TokenLe
	TokenGe
	TokenEq
	TokenNe
	TokenCaret
	TokenPipe
	TokenAndAnd
	TokenOrOr
	TokenQuestion
	TokenColon
	TokenSemicolon
	TokenAssign
	TokenStarAssign
	TokenSlashAssign
	TokenPercentAssign
	TokenPlusAssign
	TokenMinusAssign
	TokenShiftLeftAssign
	TokenShiftRightAssign
	TokenAmpAssign
	TokenCaretAssign
	TokenPipeAssign
	TokenComma

	tokenPreprocStart
	tokenPreProcGlue

	// token used privately so punctuation parsing table could be used for comments
	tokenCommentSingle
	tokenCommentMulti

	// elipsis ... is nasty, becase in moment we know .. is not followed by third dot
	// we cannot emit two tokens at once
	// we push this edge case to preprocesor that can emit multiple tokens
	tokenDots

	// another token not stricte nessesary but helps in preprocesing
	// preprocesor catches line/column on start of parsing
	tokenWhitespace

	TokenEOF
	TokenError
)

type Token struct {
	Type TokenType
	Raw  []byte

	Line, Column int
}

func (t Token) IsValid() bool {
	return t.Type != tokenNil
}

type tokenBuildFn func([]byte)

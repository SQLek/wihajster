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

	tokenPreprocStart
	tokenPreProcGlue

	// untill parser is designed, we don't know what punctuations to separate
	tokenPunctuationTBD

	// token used privately so punctuation parsing table could be used for comments
	tokenCommentSingle
	tokenCommentMulti

	// thease tokens are used as special case to parse 3 char punctuations
	tokenShiftLeft
	tokenShiftRight

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

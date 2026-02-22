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

	TokenPunctuation

	TokenEOF
	TokenError
)

type Token struct {
	Type         TokenType
	Value        string
	Line, Column int
}

package lexer

func (p *preprocesor) handleDots() (Token, error) {
	// Very likely multiple dot tokens are invalid anyway
	// if not, better implementation will come at later milestone
	tokenStr := p.accumulatorString()
	switch tokenStr {
	case ".":
		return p.makeToken(tokenPunctuationTBD), nil
	case "...":
		return p.makeToken(tokenPunctuationTBD), nil
	default:
		return p.errorf("wanted '.' or '...', got %q", tokenStr)
	}
}

func (p *preprocesor) handleKeywordOrSubsitution() (Token, error) {
	tokenStr := p.accumulatorString()

	if subTokens, isMacro := p.macros[tokenStr]; isMacro {
		return p.handleSubstitution(subTokens)
	}

	tokenType := TokenIdentifier
	switch tokenStr {
	case "auto":
		tokenType = TokenAuto
	case "break":
		tokenType = TokenBreak
	case "case":
		tokenType = TokenCase
	case "char":
		tokenType = TokenChar
	case "const":
		tokenType = TokenConst
	case "continue":
		tokenType = TokenContinue
	case "default":
		tokenType = TokenDefault
	case "do":
		tokenType = TokenDo
	case "double":
		tokenType = TokenDouble
	case "else":
		tokenType = TokenElse
	case "enum":
		tokenType = TokenEnum
	case "extern":
		tokenType = TokenExtern
	case "float":
		tokenType = TokenFloat
	case "for":
		tokenType = TokenFor
	case "goto":
		tokenType = TokenGoto
	case "if":
		tokenType = TokenIf
	case "inline":
		tokenType = TokenInline
	case "int":
		tokenType = TokenInt
	case "long":
		tokenType = TokenLong
	case "register":
		tokenType = TokenRegister
	case "restrict":
		tokenType = TokenRestrict
	case "return":
		tokenType = TokenReturn
	case "short":
		tokenType = TokenShort
	case "signed":
		tokenType = TokenSigned
	case "sizeof":
		tokenType = TokenSizeof
	case "static":
		tokenType = TokenStatic
	case "struct":
		tokenType = TokenStruct
	case "switch":
		tokenType = TokenSwitch
	case "typedef":
		tokenType = TokenTypedef
	case "union":
		tokenType = TokenUnion
	case "unsigned":
		tokenType = TokenUnsigned
	case "void":
		tokenType = TokenVoid
	case "volatile":
		tokenType = TokenVolatile
	case "while":
		tokenType = TokenWhile
	case "_Bool":
		tokenType = Token_Bool
	case "_Complex":
		tokenType = Token_Complex
	case "_Imaginary":
		tokenType = Token_Imaginary
	}

	return p.makeToken(tokenType), nil
}

func (p *preprocesor) handleSubstitution(subTokens []Token) (Token, error) {
	switch l := len(subTokens); l {
	case 0:
		// macro can be empty (header guard)
		return p.next()
	case 1:
		return subTokens[0], nil

	default:
		p.readyTokens = subTokens[1:]
		return subTokens[0], nil
	}
}

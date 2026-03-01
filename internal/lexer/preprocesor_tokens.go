package lexer

func (p *preprocesor) handleDots() (Token, error) {

	return Token{}, nil
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

		// rest of keywords

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

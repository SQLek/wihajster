package lexer

type preprocesor struct {
	// in future milestone, this will be stack of scanners
	s *scanner

	// fs.FS in future

	// macro substitution can produce more than one token
	// tokenDots can produce more than one token
	readyTokens []Token

	// accumulator for forming tokens
	accumulator []byte
}

func newPreprocesor(s *scanner) *preprocesor {
	return &preprocesor{
		s: s,
	}
}

func (p *preprocesor) next() (Token, error) {
	if tok, ok := p.popFromReady(); ok {
		return tok, nil
	}

	var line, column = p.s.line, p.s.column

	tokType, err := lex(p.s, p.tokenBuildFn)
	if err != nil {
		return Token{}, err
	}
	switch tokType {
	case tokenPreprocStart:
		return p.handleDirective()

	case TokenIdentifier:
		return p.handleKeywordOrSubsitution()

		// TODO handle dots and %:%:

	default:
		// spearate into helper method
		tok := Token{
			Type:   tokType,
			Line:   line,
			Column: column,
			Raw:    p.accumulator,
		}
		// emptying accumulator without releasing backing array
		p.accumulator = p.accumulator[:0]
		return tok, nil
	}
}

func (p *preprocesor) handleDirective() (Token, error) {

	return p.next()
}

func (p *preprocesor) handleKeywordOrSubsitution() (Token, error) {

	return Token{}, nil
}

func (p *preprocesor) tokenBuildFn(data []byte) {
	// in future milesone there will be state when we will ignore this data
	// mostly in if/ifdef/ifndef situation
	p.accumulator = append(p.accumulator, data...)
}

func (p *preprocesor) popFromReady() (Token, bool) {
	switch len(p.readyTokens) {
	case 0:
		return Token{}, false
	case 1:
		tok := p.readyTokens[0]
		p.readyTokens = nil
		return tok, true
	default:
		tok := p.readyTokens[0]
		p.readyTokens = p.readyTokens[1:]
		return tok, true
	}
}

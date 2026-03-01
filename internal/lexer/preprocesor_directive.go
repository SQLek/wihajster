package lexer

func (p *preprocesor) handleDirective() (Token, error) {
	p.ppLine = p.s.line
	tok, sameLine, err := p.nextPPToken()
	if err != nil {
		return Token{}, err
	}
	if !sameLine {
		return p.errorf("empty directive")
	}
	if tok.Type != TokenIdentifier {
		return p.errorf("expected directive name")
	}

	switch tokStr := string(tok.Raw); tokStr {
	case "define":
		return p.handleDefineName()
	default:
		return p.errorf("unsupported directive %s", tokStr)
	}
}

func (p *preprocesor) handleDefineName() (Token, error) {
	tok, sameLine, err := p.nextPPToken()
	if err != nil {
		return Token{}, err
	}
	if !sameLine || tok.Type != TokenIdentifier {
		return p.errorf("expected definition name")
	}

	macroName := string(tok.Raw)
	var macroBody []Token

	for {
		tok, sameLine, err := p.nextPPToken()
		if err != nil {
			return Token{}, err
		}
		if !sameLine {
			p.setMacro(macroName, macroBody)
			return tok, nil
		}
		macroBody = append(macroBody, tok)
	}
}

func (p *preprocesor) nextPPToken() (Token, bool, error) {
	tok, err := p.next()
	if err != nil {
		return Token{}, false, err
	}
	return tok, tok.Line == p.ppLine, nil
}

func (p *preprocesor) setMacro(name string, body []Token) {
	p.macros[name] = body
}

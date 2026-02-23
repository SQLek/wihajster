package lexer

import "fmt"

type preprocessor struct {
	tokenizer interface {
		next() (Token, error)
	}

	// tokenizer starts at 1, empty value 0 will fire new line at first token
	lastObservedLine int

	// v0 milestone only basic substitution
	macro map[string][]Token

	// one macro can turn into many tokens
	readyTokens []Token
}

func (p *preprocessor) next() (Token, error) {
	if token, ok := p.popFirstReady(); ok {
		return token, nil
	}

	token, err := p.tokenizer.next()
	if err != nil {
		return Token{}, fmt.Errorf("tokenizer: %w", err)
	}

	if p.isPrprocesorDirective(token) {
		err = p.handlePreprocesorDirective()
		if err != nil {
			return Token{}, fmt.Errorf("preprocesor: %w", err)
		}
		// preprocesor directives don't return tokens itself
		return p.next()
	}

	if token.Type == TokenIdentifier {
		return p.expandMacroOrKeyword(token)
	}

	// anything else returned as is
	return token, nil
}

func (p *preprocessor) isPrprocesorDirective(token Token) bool {
	if p.lastObservedLine == token.Line {
		// not first token in line, not a preprocesor directive
		return false
	}
	p.lastObservedLine = token.Line

	if token.Type == TokenPunctuation && token.Value == "#" {
		// '#' punctuation as a first token in line
		return true
	}

	return false
}

func (p *preprocessor) popFirstReady() (Token, bool) {
	if len(p.readyTokens) == 0 {
		return Token{}, false
	}
	token := p.readyTokens[0]

	if len(p.readyTokens) > 1 {
		p.readyTokens = p.readyTokens[1:]
	} else {
		p.readyTokens = nil
	}

	return token, true
}

func (p *preprocessor) expandMacroOrKeyword(token Token) (Token, error) {

	return token, nil
}

func (p *preprocessor) handlePreprocesorDirective() error {
	// we have read '#', fetching next
	token, err := p.next()
	if err != nil {
		return err
	}

	err = expectToken(token, TokenIdentifier, "define", "")
	if err != nil {
		return err
	}
	// #define read, now macro name

	token, err = p.next()
	if err != nil {
		return err
	}
	if token.Type != TokenIdentifier {
		return fmt.Errorf("expected defnition name, got %v", token)
	}

	macroName := token.Value
	macroLine := token.Line

	// do while to end of line
	// special condition for first token
	token, err = p.next()
	if err != nil {
		return err
	}

	return nil
}

func expectToken(token Token, kind TokenType, value string, errPrefix string) error {
	if token.Type != kind {
		return fmt.Errorf(
			"unexpected token in preprocesor directive %v at %d:%d",
			token, token.Line, token.Column,
		)
	}

	if token.Value != value {
		return fmt.Errorf(
			"only #define directive supported. got #%s%s at %d:%d",
			errPrefix, token.Value, token.Line, token.Column,
		)
	}

	return nil
}

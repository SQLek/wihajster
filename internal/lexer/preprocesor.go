package lexer

import (
	"bytes"
	"fmt"
)

type preprocesor struct {
	// in future milestone, this will be stack of scanners
	s *scanner

	// fs.FS in future

	// macro substitution can produce more than one token
	// tokenDots can produce more than one token
	readyTokens []Token

	// accumulator for forming tokens
	accumulator []byte

	macros map[string][]Token

	// macro directive only fire at start of text line ignoring whitespaces
	// to not bother with many unnesesary tokens we store on what line ended last seen token
	// we also store line on which preprocesor directive ocured
	lastSeenLine, ppLine int

	// we capturing token start from scanner
	line, column int
}

func newPreprocesor(s *scanner) *preprocesor {
	return &preprocesor{
		s:            s,
		line:         s.line,
		column:       s.column,
		lastSeenLine: s.line - 1,
		macros:       make(map[string][]Token),
	}
}

func (p *preprocesor) next() (Token, error) {
	if tok, ok := p.popFromReady(); ok {
		return tok, nil
	}

	tokType, err := p.lex()
	if err != nil {
		return Token{}, err
	}

	switch tokType {
	case tokenWhitespace:
		return p.next()
	case TokenIdentifier:
		return p.handleKeywordOrSubsitution()
	case tokenDots:
		return p.handleDots()
	case tokenPreprocStart:
		if p.lastSeenLine >= p.line {
			return p.errorf("preprocesor directive not on line start")
		}
		return p.handleDirective()
	}

	return p.makeToken(tokType), nil
}

func (p *preprocesor) errorf(format string, args ...any) (Token, error) {
	format = "%d:%d " + format
	args = append([]any{
		p.line, p.column,
	}, args...)
	return Token{}, fmt.Errorf(format, args...)
}

func (p *preprocesor) lex() (TokenType, error) {
	p.line = p.s.line
	p.column = p.s.column

	p.clearAccumulator()
	return lex(p.s, p.tokenBuildFn)
}

func (p *preprocesor) makeToken(tokType TokenType) Token {
	tok := Token{
		Type:   tokType,
		Line:   p.line,
		Column: p.column,
		Raw:    bytes.Clone(p.accumulator),
	}
	p.lastSeenLine = p.s.line
	return tok
}

func (p *preprocesor) clearAccumulator() {
	p.accumulator = p.accumulator[:0]
}

func (p *preprocesor) accumulatorString() string {
	return string(p.accumulator)
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

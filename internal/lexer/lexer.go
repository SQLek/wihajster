package lexer

import (
	"errors"
	"io/fs"
	"sync"
)

var (
	ErrNotImplementedInV0 = errors.New("feature not implemented in v0 c subset")
)

type Lexer struct {
	mx sync.Mutex

	s *scanner
	p *preprocesor

	nextTok Token
}

func NewLexer(fd fs.File /* fs.FS for include in future */) *Lexer {
	s := newScanner(fd, 0)
	p := newPreprocesor(s)
	return &Lexer{
		s: s,
		p: p,
	}
}

func (l *Lexer) Peek() (Token, error) {
	if tok := l.nextTok; tok.Type != tokenNil {
		return tok, nil
	}

	tok, err := l.p.next()
	if err != nil {
		return Token{}, err
	}

	l.nextTok = tok
	return tok, nil
}

func (l *Lexer) Next() (Token, error) {
	if tok := l.nextTok; tok.Type != tokenNil {
		l.nextTok = Token{}
		return tok, nil
	}

	return l.p.next()
}

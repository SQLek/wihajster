package lexer

import (
	"io"
	"sync"
)

type Lexer struct {
	mx sync.Mutex

	tokenizer    *tokenizer
	preprocessor *preprocessor

	peekedToken Token
}

func New(scanner io.ByteScanner) *Lexer {
	tokenizer := &tokenizer{scanner: scanner, line: 1, column: 0}
	preprocessor := &preprocessor{tokenizer: tokenizer}
	return &Lexer{
		tokenizer:    tokenizer,
		preprocessor: preprocessor,
	}
}

func (l *Lexer) Next() (Token, error) {
	l.mx.Lock()
	defer l.mx.Unlock()

	if l.peekedToken.Type == tokenNil {
		return l.preprocessor.next()
	}

	token := l.peekedToken
	l.peekedToken = Token{}
	return token, nil
}

func (l *Lexer) Peek() (Token, error) {
	l.mx.Lock()
	defer l.mx.Unlock()

	if l.peekedToken.Type != tokenNil {
		return l.peekedToken, nil
	}

	token, err := l.preprocessor.next()
	if err != nil {
		return Token{}, err
	}
	l.peekedToken = token
	return token, nil
}

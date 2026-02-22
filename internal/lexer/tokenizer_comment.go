package lexer

import "errors"

func (l *tokenizer) readSingleLineComment() (Token, error) {
	// we know that "//" has already been read
	value, err := charClassNewLine.collectNegatingFrom(l.scanner)
	if err != nil {
		return Token{}, err
	}

	// edge case there is \\\n in the comment,
	// which means that the comment continues on the next line
	if len(value) > 0 && value[len(value)-1] == '\\' {
		l.line++
		return l.readSingleLineComment()
	}

	return l.next()
}

func (l *tokenizer) readMultiLineComment() (Token, error) {
	return Token{Type: TokenError}, errors.New("not implemented")
}

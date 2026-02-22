package lexer

type preprocessor struct {
	tokenizer *tokenizer
}

func (p *preprocessor) next() (Token, error) {
	// TODO: implement preprocessor directives, for now just pass through tokens from tokenizer
	return p.tokenizer.next()
}

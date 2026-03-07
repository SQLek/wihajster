package parser

import (
	"errors"
	"fmt"
	"io"

	"github.com/SQLek/wihajster/internal/lexer"
)

type tokenSource interface {
	Peek() (lexer.Token, error)
	Next() (lexer.Token, error)
}

type Parser struct {
	tokens tokenSource

	fatalLexErr error
	diagnostics []*Error

	last lexer.Token
}

func New(tokens tokenSource) *Parser {
	return &Parser{tokens: tokens}
}

func Parse(tokens tokenSource) (*TranslationUnit, error) {
	return New(tokens).ParseTranslationUnit()
}

func (p *Parser) ParseTranslationUnit() (*TranslationUnit, error) {
	tu := &TranslationUnit{}
	for {
		tok := p.peekTok()
		if tok.Type == lexer.TokenEOF {
			break
		}

		fn, ok := p.parseFunctionDefinition()
		if ok {
			tu.Functions = append(tu.Functions, fn)
			continue
		}

		if p.peekTok().Type == lexer.TokenEOF {
			break
		}
		p.synchronizeTopLevel()
	}

	if err := p.finishError(); err != nil {
		return nil, err
	}
	return tu, nil
}

func (p *Parser) parseFunctionDefinition() (FunctionDefinition, bool) {
	typeTok, typ, ok := p.parseTypeSpecifier()
	if !ok {
		return FunctionDefinition{}, false
	}

	if p.peekTok().Type == lexer.TokenStar {
		p.addDiagnostic(unsupportedError(p.peekTok(), "pointers"))
		p.nextTok()
		return FunctionDefinition{}, false
	}

	_, name, ok := p.expectIdent("function name")
	if !ok {
		return FunctionDefinition{}, false
	}

	if !p.expectToken(lexer.TokenLParen, "'('") {
		return FunctionDefinition{}, false
	}

	if tok := p.peekTok(); tok.Type != lexer.TokenRParen {
		p.addDiagnostic(unsupportedError(tok, "function parameters"))
		for tok.Type != lexer.TokenRParen && tok.Type != lexer.TokenEOF {
			p.nextTok()
			tok = p.peekTok()
		}
	}
	if !p.expectToken(lexer.TokenRParen, "')'") {
		return FunctionDefinition{}, false
	}

	body, ok := p.parseBlockStatement()
	if !ok {
		return FunctionDefinition{}, false
	}

	return FunctionDefinition{
		Token:      typeTok,
		ReturnType: typ,
		Name:       name,
		Body:       body,
	}, true
}

func (p *Parser) parseTypeSpecifier() (lexer.Token, TypeSpecifier, bool) {
	tok := p.peekTok()

	switch tok.Type {
	case lexer.TokenInt:
		p.nextTok()
		return tok, TypeSpecifierInt, true
	case lexer.TokenVoid:
		p.nextTok()
		return tok, TypeSpecifierVoid, true
	case lexer.TokenStruct:
		p.addDiagnostic(unsupportedError(tok, "struct declarations"))
		p.nextTok()
		return lexer.Token{}, 0, false
	default:
		p.addDiagnostic(newError(p.errorToken(tok), "expected type specifier, got %s", tokenDescription(tok)))
		return lexer.Token{}, 0, false
	}
}

func (p *Parser) parseStatement() (Statement, bool) {
	tok := p.peekTok()

	switch tok.Type {
	case lexer.TokenLBrace:
		stmt, ok := p.parseBlockStatement()
		if !ok {
			return nil, false
		}
		return stmt, true
	case lexer.TokenReturn:
		return p.parseReturnStatement()
	case lexer.TokenIf:
		return p.parseIfStatement()
	case lexer.TokenWhile:
		return p.parseWhileStatement()
	case lexer.TokenInt, lexer.TokenVoid:
		p.addDiagnostic(unsupportedError(tok, "declarations beyond current subset"))
		return nil, false
	case lexer.TokenStruct:
		p.addDiagnostic(unsupportedError(tok, "struct declarations"))
		return nil, false
	default:
		return p.parseExpressionStatement()
	}
}

func (p *Parser) parseBlockStatement() (BlockStatement, bool) {
	open, ok := p.expect(lexer.TokenLBrace, "'{'")
	if !ok {
		return BlockStatement{}, false
	}

	var stmts []Statement
	for {
		tok := p.peekTok()
		switch tok.Type {
		case lexer.TokenEOF:
			p.addDiagnostic(newError(open, "expected '}' before end of file"))
			return BlockStatement{}, false
		case lexer.TokenRBrace:
			p.nextTok()
			return BlockStatement{Token: open, Statements: stmts}, true
		}

		stmt, stmtOK := p.parseStatement()
		if stmtOK {
			stmts = append(stmts, stmt)
			continue
		}
		p.synchronizeStatement()
	}
}

func (p *Parser) parseReturnStatement() (Statement, bool) {
	retTok, ok := p.expect(lexer.TokenReturn, "'return'")
	if !ok {
		return nil, false
	}

	if p.accept(lexer.TokenSemicolon) {
		return ReturnStatement{Token: retTok}, true
	}

	expr, ok := p.parseExpression(0)
	if !ok {
		return nil, false
	}
	if !p.expectToken(lexer.TokenSemicolon, "';'") {
		return nil, false
	}
	return ReturnStatement{Token: retTok, Expression: expr}, true
}

func (p *Parser) parseIfStatement() (Statement, bool) {
	ifTok, ok := p.expect(lexer.TokenIf, "'if'")
	if !ok {
		return nil, false
	}
	if !p.expectToken(lexer.TokenLParen, "'('") {
		return nil, false
	}
	cond, ok := p.parseExpression(0)
	if !ok {
		return nil, false
	}
	if !p.expectToken(lexer.TokenRParen, "')'") {
		return nil, false
	}
	thenStmt, ok := p.parseStatement()
	if !ok {
		return nil, false
	}

	if p.accept(lexer.TokenElse) {
		elseStmt, ok := p.parseStatement()
		if !ok {
			return nil, false
		}
		return IfStatement{Token: ifTok, Cond: cond, Then: thenStmt, Else: elseStmt}, true
	}

	return IfStatement{Token: ifTok, Cond: cond, Then: thenStmt}, true
}

func (p *Parser) parseWhileStatement() (Statement, bool) {
	whileTok, ok := p.expect(lexer.TokenWhile, "'while'")
	if !ok {
		return nil, false
	}
	if !p.expectToken(lexer.TokenLParen, "'('") {
		return nil, false
	}
	cond, ok := p.parseExpression(0)
	if !ok {
		return nil, false
	}
	if !p.expectToken(lexer.TokenRParen, "')'") {
		return nil, false
	}
	body, ok := p.parseStatement()
	if !ok {
		return nil, false
	}

	return WhileStatement{Token: whileTok, Cond: cond, Body: body}, true
}

func (p *Parser) parseExpressionStatement() (Statement, bool) {
	tok := p.peekTok()

	if p.accept(lexer.TokenSemicolon) {
		return ExpressionStatement{Token: tok}, true
	}

	expr, ok := p.parseExpression(0)
	if !ok {
		return nil, false
	}
	if !p.expectToken(lexer.TokenSemicolon, "';'") {
		return nil, false
	}
	return ExpressionStatement{Token: tok, Expression: expr}, true
}

func (p *Parser) parseExpression(minPrec int) (Expression, bool) {
	lhs, ok := p.parseUnaryExpression()
	if !ok {
		return nil, false
	}

	for {
		tok := p.peekTok()
		prec := infixPrecedence(tok.Type)
		if prec < minPrec {
			return lhs, true
		}

		p.nextTok()
		rhs, ok := p.parseExpression(prec + 1)
		if !ok {
			return nil, false
		}
		lhs = BinaryExpression{Token: tok, Op: tok.Type, LHS: lhs, RHS: rhs}
	}
}

func (p *Parser) parseUnaryExpression() (Expression, bool) {
	tok := p.peekTok()

	switch tok.Type {
	case lexer.TokenMinus, lexer.TokenBang, lexer.TokenTilde, lexer.TokenPlus:
		p.nextTok()
		op, ok := p.parseUnaryExpression()
		if !ok {
			return nil, false
		}
		return UnaryExpression{Token: tok, Op: tok.Type, Operand: op}, true
	default:
		return p.parsePrimaryExpression()
	}
}

func (p *Parser) parsePrimaryExpression() (Expression, bool) {
	tok := p.peekTok()

	switch tok.Type {
	case lexer.TokenIdentifier:
		tok, name, ok := p.expectIdent("identifier")
		if !ok {
			return nil, false
		}
		return IdentifierExpression{Token: tok, Name: name}, true
	case lexer.TokenIntegerConstant:
		tok, raw, ok := p.expectInteger("integer literal")
		if !ok {
			return nil, false
		}
		return IntegerLiteralExpression{Token: tok, Raw: raw}, true
	case lexer.TokenLParen:
		p.nextTok()
		expr, ok := p.parseExpression(0)
		if !ok {
			return nil, false
		}
		if !p.expectToken(lexer.TokenRParen, "')'") {
			return nil, false
		}
		return expr, true
	case lexer.TokenStar:
		p.addDiagnostic(unsupportedError(tok, "pointers"))
		return nil, false
	default:
		p.addDiagnostic(newError(p.errorToken(tok), "expected expression, got %s", tokenDescription(tok)))
		return nil, false
	}
}

func infixPrecedence(tt lexer.TokenType) int {
	switch tt {
	case lexer.TokenOrOr:
		return 1
	case lexer.TokenAndAnd:
		return 2
	case lexer.TokenPipe:
		return 3
	case lexer.TokenCaret:
		return 4
	case lexer.TokenAmp:
		return 5
	case lexer.TokenEq, lexer.TokenNe:
		return 6
	case lexer.TokenLt, lexer.TokenLe, lexer.TokenGt, lexer.TokenGe:
		return 7
	case lexer.TokenShiftLeft, lexer.TokenShiftRight:
		return 8
	case lexer.TokenPlus, lexer.TokenMinus:
		return 9
	case lexer.TokenStar, lexer.TokenSlash, lexer.TokenPercent:
		return 10
	default:
		return -1
	}
}

func (p *Parser) expectToken(tt lexer.TokenType, what string) bool {
	_, ok := p.expect(tt, what)
	return ok
}

func (p *Parser) expect(tt lexer.TokenType, what string) (lexer.Token, bool) {
	tok := p.nextTok()
	if tok.Type != tt {
		if tok.Type == lexer.TokenEOF {
			p.addDiagnostic(newError(p.errorToken(tok), "unexpected end of file, expected %s", what))
		} else {
			p.addDiagnostic(newError(tok, "expected %s, got %s", what, tokenDescription(tok)))
		}
		return lexer.Token{}, false
	}
	return tok, true
}

func (p *Parser) expectIdent(what string) (lexer.Token, string, bool) {
	tok, ok := p.expect(lexer.TokenIdentifier, what)
	if !ok {
		return lexer.Token{}, "", false
	}
	return tok, string(tok.Raw), true
}

func (p *Parser) expectInteger(what string) (lexer.Token, string, bool) {
	tok, ok := p.expect(lexer.TokenIntegerConstant, what)
	if !ok {
		return lexer.Token{}, "", false
	}
	return tok, string(tok.Raw), true
}

func (p *Parser) accept(tt lexer.TokenType) bool {
	if p.peekTok().Type != tt {
		return false
	}
	p.nextTok()
	return true
}

func (p *Parser) peekTok() lexer.Token {
	tok, err := p.tokens.Peek()
	tok = p.normalizeToken(tok, err)
	if tok.Type != lexer.TokenEOF {
		p.last = tok
	}
	return tok
}

func (p *Parser) nextTok() lexer.Token {
	tok, err := p.tokens.Next()
	tok = p.normalizeToken(tok, err)
	if tok.Type != lexer.TokenEOF {
		p.last = tok
	}
	return tok
}

func (p *Parser) normalizeToken(tok lexer.Token, err error) lexer.Token {
	if err != nil {
		if !errors.Is(err, io.EOF) && p.fatalLexErr == nil {
			p.fatalLexErr = err
		}
		return lexer.Token{Type: lexer.TokenEOF}
	}
	if tok.Type == lexer.TokenError {
		if p.fatalLexErr == nil {
			p.fatalLexErr = fmt.Errorf("%s", string(tok.Raw))
		}
		return lexer.Token{Type: lexer.TokenEOF}
	}
	if tok.Type == 0 {
		return lexer.Token{Type: lexer.TokenEOF}
	}
	return tok
}

func (p *Parser) addDiagnostic(err *Error) {
	if err != nil {
		p.diagnostics = append(p.diagnostics, err)
	}
}

func (p *Parser) errorToken(tok lexer.Token) lexer.Token {
	if tok.Type == lexer.TokenEOF && p.last.IsValid() {
		return p.last
	}
	return tok
}

func (p *Parser) synchronizeStatement() {
	for {
		tok := p.peekTok()
		switch tok.Type {
		case lexer.TokenEOF, lexer.TokenRBrace:
			return
		case lexer.TokenSemicolon:
			p.nextTok()
			return
		default:
			p.nextTok()
		}
	}
}

func (p *Parser) synchronizeTopLevel() {
	for {
		tok := p.peekTok()
		switch tok.Type {
		case lexer.TokenEOF, lexer.TokenInt, lexer.TokenVoid:
			return
		default:
			p.nextTok()
		}
	}
}

func (p *Parser) finishError() error {
	if p.fatalLexErr == nil && len(p.diagnostics) == 0 {
		return nil
	}
	return &ParseErrors{
		FatalLexer:  p.fatalLexErr,
		Diagnostics: p.diagnostics,
	}
}

func tokenDescription(tok lexer.Token) string {
	if len(tok.Raw) > 0 {
		return fmt.Sprintf("%q", string(tok.Raw))
	}
	if tok.Type == lexer.TokenEOF {
		return "end of file"
	}
	return fmt.Sprintf("token(%d)", tok.Type)
}

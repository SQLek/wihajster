package parser

import (
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
	last   lexer.Token
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
		tok, err := p.peek()
		if err == io.EOF {
			return tu, nil
		}
		if err != nil {
			return nil, err
		}

		if tok.Type == lexer.TokenEOF {
			_, _ = p.next()
			return tu, nil
		}

		fn, err := p.parseFunctionDefinition()
		if err != nil {
			return nil, err
		}
		tu.Functions = append(tu.Functions, fn)
	}
}

func (p *Parser) parseFunctionDefinition() (FunctionDefinition, error) {
	typeTok, typ, err := p.parseTypeSpecifier()
	if err != nil {
		return FunctionDefinition{}, err
	}

	if tok, err := p.peek(); err == nil && tok.Type == lexer.TokenStar {
		return FunctionDefinition{}, unsupportedError(tok, "pointers")
	}

	name, err := p.expect(lexer.TokenIdentifier, "function name")
	if err != nil {
		return FunctionDefinition{}, err
	}

	if _, err := p.expect(lexer.TokenLParen, "'('"); err != nil {
		return FunctionDefinition{}, err
	}
	if tok, err := p.peek(); err != nil {
		return FunctionDefinition{}, err
	} else if tok.Type != lexer.TokenRParen {
		return FunctionDefinition{}, unsupportedError(tok, "function parameters")
	}
	if _, err := p.expect(lexer.TokenRParen, "')'"); err != nil {
		return FunctionDefinition{}, err
	}

	body, err := p.parseBlockStatement()
	if err != nil {
		return FunctionDefinition{}, err
	}

	return FunctionDefinition{
		Token:      typeTok,
		ReturnType: typ,
		Name:       string(name.Raw),
		Body:       body,
	}, nil
}

func (p *Parser) parseTypeSpecifier() (lexer.Token, TypeSpecifier, error) {
	tok, err := p.peek()
	if err != nil {
		return lexer.Token{}, 0, err
	}

	switch tok.Type {
	case lexer.TokenInt:
		_, _ = p.next()
		return tok, TypeSpecifierInt, nil
	case lexer.TokenVoid:
		_, _ = p.next()
		return tok, TypeSpecifierVoid, nil
	case lexer.TokenStruct:
		return lexer.Token{}, 0, unsupportedError(tok, "struct declarations")
	default:
		return lexer.Token{}, 0, newError(tok, "expected type specifier, got %s", tokenDescription(tok))
	}
}

func (p *Parser) parseStatement() (Statement, error) {
	tok, err := p.peek()
	if err != nil {
		return nil, err
	}

	switch tok.Type {
	case lexer.TokenLBrace:
		stmt, err := p.parseBlockStatement()
		if err != nil {
			return nil, err
		}
		return stmt, nil
	case lexer.TokenReturn:
		return p.parseReturnStatement()
	case lexer.TokenIf:
		return p.parseIfStatement()
	case lexer.TokenWhile:
		return p.parseWhileStatement()
	case lexer.TokenInt, lexer.TokenVoid:
		return nil, unsupportedError(tok, "declarations beyond current subset")
	case lexer.TokenStruct:
		return nil, unsupportedError(tok, "struct declarations")
	default:
		return p.parseExpressionStatement()
	}
}

func (p *Parser) parseBlockStatement() (BlockStatement, error) {
	open, err := p.expect(lexer.TokenLBrace, "'{'")
	if err != nil {
		return BlockStatement{}, err
	}

	var stmts []Statement
	for {
		tok, err := p.peek()
		if err != nil {
			if err == io.EOF {
				return BlockStatement{}, newError(open, "expected '}' before end of file")
			}
			return BlockStatement{}, err
		}
		if tok.Type == lexer.TokenRBrace {
			_, _ = p.next()
			break
		}
		stmt, err := p.parseStatement()
		if err != nil {
			return BlockStatement{}, err
		}
		stmts = append(stmts, stmt)
	}

	return BlockStatement{Token: open, Statements: stmts}, nil
}

func (p *Parser) parseReturnStatement() (Statement, error) {
	retTok, err := p.expect(lexer.TokenReturn, "'return'")
	if err != nil {
		return nil, err
	}

	tok, err := p.peek()
	if err != nil {
		return nil, err
	}

	if tok.Type == lexer.TokenSemicolon {
		_, _ = p.next()
		return ReturnStatement{Token: retTok}, nil
	}

	expr, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(lexer.TokenSemicolon, "';'"); err != nil {
		return nil, err
	}
	return ReturnStatement{Token: retTok, Expression: expr}, nil
}

func (p *Parser) parseIfStatement() (Statement, error) {
	ifTok, err := p.expect(lexer.TokenIf, "'if'")
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(lexer.TokenLParen, "'('"); err != nil {
		return nil, err
	}
	cond, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(lexer.TokenRParen, "')'"); err != nil {
		return nil, err
	}
	thenStmt, err := p.parseStatement()
	if err != nil {
		return nil, err
	}

	tok, err := p.peek()
	if err != nil && err != io.EOF {
		return nil, err
	}
	if err == nil && tok.Type == lexer.TokenElse {
		_, _ = p.next()
		elseStmt, err := p.parseStatement()
		if err != nil {
			return nil, err
		}
		return IfStatement{Token: ifTok, Cond: cond, Then: thenStmt, Else: elseStmt}, nil
	}

	return IfStatement{Token: ifTok, Cond: cond, Then: thenStmt}, nil
}

func (p *Parser) parseWhileStatement() (Statement, error) {
	whileTok, err := p.expect(lexer.TokenWhile, "'while'")
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(lexer.TokenLParen, "'('"); err != nil {
		return nil, err
	}
	cond, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(lexer.TokenRParen, "')'"); err != nil {
		return nil, err
	}
	body, err := p.parseStatement()
	if err != nil {
		return nil, err
	}

	return WhileStatement{Token: whileTok, Cond: cond, Body: body}, nil
}

func (p *Parser) parseExpressionStatement() (Statement, error) {
	tok, err := p.peek()
	if err != nil {
		return nil, err
	}

	if tok.Type == lexer.TokenSemicolon {
		_, _ = p.next()
		return ExpressionStatement{Token: tok}, nil
	}

	expr, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(lexer.TokenSemicolon, "';'"); err != nil {
		return nil, err
	}
	return ExpressionStatement{Token: tok, Expression: expr}, nil
}

func (p *Parser) parseExpression(minPrec int) (Expression, error) {
	lhs, err := p.parseUnaryExpression()
	if err != nil {
		return nil, err
	}

	for {
		tok, err := p.peek()
		if err != nil {
			if err == io.EOF {
				return lhs, nil
			}
			return nil, err
		}
		prec := infixPrecedence(tok.Type)
		if prec < minPrec {
			return lhs, nil
		}

		_, _ = p.next()
		rhs, err := p.parseExpression(prec + 1)
		if err != nil {
			return nil, err
		}
		lhs = BinaryExpression{Token: tok, Op: tok.Type, LHS: lhs, RHS: rhs}
	}
}

func (p *Parser) parseUnaryExpression() (Expression, error) {
	tok, err := p.peek()
	if err != nil {
		return nil, err
	}

	switch tok.Type {
	case lexer.TokenMinus, lexer.TokenBang, lexer.TokenTilde, lexer.TokenPlus:
		_, _ = p.next()
		op, err := p.parseUnaryExpression()
		if err != nil {
			return nil, err
		}
		return UnaryExpression{Token: tok, Op: tok.Type, Operand: op}, nil
	default:
		return p.parsePrimaryExpression()
	}
}

func (p *Parser) parsePrimaryExpression() (Expression, error) {
	tok, err := p.peek()
	if err != nil {
		return nil, err
	}

	switch tok.Type {
	case lexer.TokenIdentifier:
		_, _ = p.next()
		return IdentifierExpression{Token: tok, Name: string(tok.Raw)}, nil
	case lexer.TokenIntegerConstant:
		_, _ = p.next()
		return IntegerLiteralExpression{Token: tok, Raw: string(tok.Raw)}, nil
	case lexer.TokenLParen:
		_, _ = p.next()
		expr, err := p.parseExpression(0)
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(lexer.TokenRParen, "')'"); err != nil {
			return nil, err
		}
		return expr, nil
	case lexer.TokenStar:
		return nil, unsupportedError(tok, "pointers")
	default:
		return nil, newError(tok, "expected expression, got %s", tokenDescription(tok))
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

func (p *Parser) expect(tt lexer.TokenType, what string) (lexer.Token, error) {
	tok, err := p.next()
	if err != nil {
		if err == io.EOF {
			return lexer.Token{}, newError(p.last, "unexpected end of file, expected %s", what)
		}
		return lexer.Token{}, err
	}
	if tok.Type != tt {
		return lexer.Token{}, newError(tok, "expected %s, got %s", what, tokenDescription(tok))
	}
	return tok, nil
}

func (p *Parser) peek() (lexer.Token, error) {
	tok, err := p.tokens.Peek()
	if err == io.EOF {
		return tok, io.EOF
	}
	if err != nil {
		return lexer.Token{}, err
	}
	if tok.Type == lexer.TokenError {
		return lexer.Token{}, newError(tok, "%s", string(tok.Raw))
	}
	p.last = tok
	return tok, nil
}

func (p *Parser) next() (lexer.Token, error) {
	tok, err := p.tokens.Next()
	if err == io.EOF {
		return tok, io.EOF
	}
	if err != nil {
		return lexer.Token{}, err
	}
	if tok.Type == lexer.TokenError {
		return lexer.Token{}, newError(tok, "%s", string(tok.Raw))
	}
	return tok, nil
}

func tokenDescription(tok lexer.Token) string {
	if len(tok.Raw) > 0 {
		return fmt.Sprintf("%q", string(tok.Raw))
	}
	return fmt.Sprintf("token(%d)", tok.Type)
}

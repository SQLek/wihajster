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

		fn, decl, ok := p.parseExternalDeclaration()
		if ok {
			if fn != nil {
				tu.Functions = append(tu.Functions, *fn)
			} else {
				tu.Declarations = append(tu.Declarations, *decl)
			}
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

func (p *Parser) parseExternalDeclaration() (*FunctionDefinition, *Declaration, bool) {
	typeTok, typ, ok := p.parseTypeName()
	if !ok {
		return nil, nil, false
	}

	nameTok, name, ptrDepth, ok := p.parseDeclarator("declaration name")
	if !ok {
		return nil, nil, false
	}
	typ.PointerDepth += ptrDepth

	if p.accept(lexer.TokenLParen) {
		params, ok := p.parseParameterList()
		if !ok {
			return nil, nil, false
		}
		if !p.expectToken(lexer.TokenRParen, "')'") {
			return nil, nil, false
		}

		if p.accept(lexer.TokenSemicolon) {
			p.addDiagnostic(unsupportedError(nameTok, "function declarations without body"))
			return nil, nil, false
		}

		body, ok := p.parseBlockStatement()
		if !ok {
			return nil, nil, false
		}

		fn := FunctionDefinition{
			Token:      typeTok,
			ReturnType: typ,
			Name:       name,
			Parameters: params,
			Body:       body,
		}
		return &fn, nil, true
	}

	decl, ok := p.parseDeclarationTail(typeTok, typ, nameTok, name)
	if !ok {
		return nil, nil, false
	}
	return nil, &decl, true
}

func (p *Parser) parseTypeName() (lexer.Token, TypeName, bool) {
	tok := p.peekTok()

	switch tok.Type {
	case lexer.TokenInt:
		p.nextTok()
		return tok, TypeName{Token: tok, Specifier: TypeSpecifierInt}, true
	case lexer.TokenChar:
		p.nextTok()
		return tok, TypeName{Token: tok, Specifier: TypeSpecifierChar}, true
	case lexer.TokenVoid:
		p.nextTok()
		return tok, TypeName{Token: tok, Specifier: TypeSpecifierVoid}, true
	case lexer.TokenStruct:
		p.addDiagnostic(unsupportedError(tok, "struct declarations"))
		p.nextTok()
		return lexer.Token{}, TypeName{}, false
	case lexer.TokenUnion:
		p.addDiagnostic(unsupportedError(tok, "union declarations"))
		p.nextTok()
		return lexer.Token{}, TypeName{}, false
	case lexer.TokenEnum:
		p.addDiagnostic(unsupportedError(tok, "enum declarations"))
		p.nextTok()
		return lexer.Token{}, TypeName{}, false
	case lexer.TokenFloat, lexer.TokenDouble:
		p.addDiagnostic(unsupportedError(tok, "floating-point types"))
		p.nextTok()
		return lexer.Token{}, TypeName{}, false
	default:
		p.addDiagnostic(newError(p.errorToken(tok), "expected type specifier, got %s", tokenDescription(tok)))
		return lexer.Token{}, TypeName{}, false
	}
}

func (p *Parser) parseDeclarator(what string) (lexer.Token, string, int, bool) {
	ptrDepth := 0
	for p.accept(lexer.TokenStar) {
		ptrDepth++
	}

	if p.peekTok().Type == lexer.TokenLParen {
		tok := p.peekTok()
		p.addDiagnostic(unsupportedError(tok, "function pointers"))
		return lexer.Token{}, "", 0, false
	}

	tok, name, ok := p.expectIdent(what)
	if !ok {
		return lexer.Token{}, "", 0, false
	}

	if p.peekTok().Type == lexer.TokenLBracket {
		p.addDiagnostic(unsupportedError(p.peekTok(), "arrays"))
		return lexer.Token{}, "", 0, false
	}

	return tok, name, ptrDepth, true
}

func (p *Parser) parseParameterList() ([]FunctionParameter, bool) {
	if p.peekTok().Type == lexer.TokenRParen {
		return nil, true
	}

	if p.peekTok().Type == lexer.TokenEllipsis {
		p.addDiagnostic(unsupportedError(p.peekTok(), "variadic functions"))
		return nil, false
	}

	var params []FunctionParameter
	for {
		_, typ, ok := p.parseTypeName()
		if !ok {
			return nil, false
		}
		tok, name, ptrDepth, ok := p.parseDeclarator("parameter name")
		if !ok {
			return nil, false
		}
		typ.PointerDepth += ptrDepth
		if typ.Specifier == TypeSpecifierVoid && typ.PointerDepth == 0 {
			p.addDiagnostic(unsupportedError(tok, "void objects"))
			return nil, false
		}
		params = append(params, FunctionParameter{Token: tok, Type: typ, Name: name})

		if p.accept(lexer.TokenComma) {
			if p.peekTok().Type == lexer.TokenEllipsis {
				p.addDiagnostic(unsupportedError(p.peekTok(), "variadic functions"))
				return nil, false
			}
			continue
		}
		break
	}

	return params, true
}

func (p *Parser) parseDeclarationTail(typeTok lexer.Token, typ TypeName, nameTok lexer.Token, name string) (Declaration, bool) {
	decl := Declaration{Token: typeTok, Type: typ, Name: name}
	if p.accept(lexer.TokenAssign) {
		initExpr, ok := p.parseExpression(0)
		if !ok {
			return Declaration{}, false
		}
		if p.hasDisallowedExprTail() {
			return Declaration{}, false
		}
		decl.Initializer = initExpr
	}

	if p.accept(lexer.TokenComma) {
		p.addDiagnostic(unsupportedError(nameTok, "multiple declarators in one declaration"))
		return Declaration{}, false
	}

	if !p.expectToken(lexer.TokenSemicolon, "';'") {
		return Declaration{}, false
	}
	return decl, true
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
	case lexer.TokenFor:
		return p.parseForStatement()
	case lexer.TokenInt, lexer.TokenChar, lexer.TokenVoid:
		return p.parseDeclarationStatement()
	case lexer.TokenStruct:
		p.addDiagnostic(unsupportedError(tok, "struct declarations"))
		return nil, false
	case lexer.TokenSwitch:
		p.addDiagnostic(unsupportedError(tok, "switch statements"))
		return nil, false
	case lexer.TokenGoto:
		p.addDiagnostic(unsupportedError(tok, "goto statements"))
		return nil, false
	case lexer.TokenDo:
		p.addDiagnostic(unsupportedError(tok, "do-while statements"))
		return nil, false
	case lexer.TokenBreak:
		p.addDiagnostic(unsupportedError(tok, "break statements"))
		return nil, false
	case lexer.TokenContinue:
		p.addDiagnostic(unsupportedError(tok, "continue statements"))
		return nil, false
	default:
		return p.parseExpressionStatement()
	}
}

func (p *Parser) parseDeclarationStatement() (Statement, bool) {
	typeTok, typ, ok := p.parseTypeName()
	if !ok {
		return nil, false
	}
	nameTok, name, ptrDepth, ok := p.parseDeclarator("declaration name")
	if !ok {
		return nil, false
	}
	typ.PointerDepth += ptrDepth
	if typ.Specifier == TypeSpecifierVoid && typ.PointerDepth == 0 {
		p.addDiagnostic(unsupportedError(nameTok, "void objects"))
		return nil, false
	}
	decl, ok := p.parseDeclarationTail(typeTok, typ, nameTok, name)
	if !ok {
		return nil, false
	}
	return DeclarationStatement{Token: typeTok, Declaration: decl}, true
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
	if p.hasDisallowedExprTail() {
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
	if p.hasDisallowedExprTail() {
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
	if p.hasDisallowedExprTail() {
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

func (p *Parser) parseForStatement() (Statement, bool) {
	forTok, ok := p.expect(lexer.TokenFor, "'for'")
	if !ok {
		return nil, false
	}
	if !p.expectToken(lexer.TokenLParen, "'('") {
		return nil, false
	}

	var init Statement
	if p.accept(lexer.TokenSemicolon) {
		init = nil
	} else if isTypeSpecifierToken(p.peekTok().Type) {
		stmt, ok := p.parseDeclarationStatement()
		if !ok {
			return nil, false
		}
		init = stmt
	} else {
		tok := p.peekTok()
		expr, ok := p.parseExpression(0)
		if !ok {
			return nil, false
		}
		if p.hasDisallowedExprTail() {
			return nil, false
		}
		if !p.expectToken(lexer.TokenSemicolon, "';'") {
			return nil, false
		}
		init = ExpressionStatement{Token: tok, Expression: expr}
	}

	var cond Expression
	if !p.accept(lexer.TokenSemicolon) {
		parsedCond, ok := p.parseExpression(0)
		if !ok {
			return nil, false
		}
		if p.hasDisallowedExprTail() {
			return nil, false
		}
		cond = parsedCond
		if !p.expectToken(lexer.TokenSemicolon, "';'") {
			return nil, false
		}
	}

	var post Expression
	if p.peekTok().Type != lexer.TokenRParen {
		parsedPost, ok := p.parseExpression(0)
		if !ok {
			return nil, false
		}
		if p.hasDisallowedExprTail() {
			return nil, false
		}
		post = parsedPost
	}

	if !p.expectToken(lexer.TokenRParen, "')'") {
		return nil, false
	}
	body, ok := p.parseStatement()
	if !ok {
		return nil, false
	}

	return ForStatement{Token: forTok, Init: init, Cond: cond, Post: post, Body: body}, true
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
	if p.hasDisallowedExprTail() {
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
		if tok.Type == lexer.TokenAssign {
			if minPrec > 0 {
				return lhs, true
			}
			p.nextTok()
			rhs, ok := p.parseExpression(0)
			if !ok {
				return nil, false
			}
			lhs = AssignmentExpression{Token: tok, LHS: lhs, RHS: rhs}
			continue
		}

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
	case lexer.TokenMinus, lexer.TokenBang, lexer.TokenTilde, lexer.TokenPlus, lexer.TokenStar, lexer.TokenAmp:
		p.nextTok()
		op, ok := p.parseUnaryExpression()
		if !ok {
			return nil, false
		}
		return UnaryExpression{Token: tok, Op: tok.Type, Operand: op}, true
	default:
		return p.parsePostfixExpression()
	}
}

func (p *Parser) parsePostfixExpression() (Expression, bool) {
	expr, ok := p.parsePrimaryExpression()
	if !ok {
		return nil, false
	}

	for p.accept(lexer.TokenLParen) {
		callTok := p.last
		var args []Expression
		if p.peekTok().Type != lexer.TokenRParen {
			for {
				arg, ok := p.parseExpression(0)
				if !ok {
					return nil, false
				}
				if p.peekTok().Type == lexer.TokenQuestion {
					p.addDiagnostic(unsupportedError(p.peekTok(), "ternary operator"))
					return nil, false
				}
				args = append(args, arg)
				if p.accept(lexer.TokenComma) {
					continue
				}
				break
			}
		}
		if !p.expectToken(lexer.TokenRParen, "')'") {
			return nil, false
		}
		expr = CallExpression{Token: callTok, Callee: expr, Args: args}
	}

	return expr, true
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
	case lexer.TokenCharacterConstant:
		tok := p.nextTok()
		return CharacterLiteralExpression{Token: tok, Raw: string(tok.Raw)}, true
	case lexer.TokenLParen:
		openTok := p.nextTok()
		if isTypeSpecifierToken(p.peekTok().Type) {
			p.addDiagnostic(unsupportedError(openTok, "casts"))
			return nil, false
		}
		expr, ok := p.parseExpression(0)
		if !ok {
			return nil, false
		}
		if !p.expectToken(lexer.TokenRParen, "')'") {
			return nil, false
		}
		return expr, true
	case lexer.TokenPlusPlus, lexer.TokenMinusMinus:
		p.addDiagnostic(unsupportedError(tok, "increment/decrement operators"))
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

func (p *Parser) hasDisallowedExprTail() bool {
	tok := p.peekTok()
	switch tok.Type {
	case lexer.TokenQuestion:
		p.addDiagnostic(unsupportedError(tok, "ternary operator"))
		return true
	case lexer.TokenComma:
		p.addDiagnostic(unsupportedError(tok, "comma operator"))
		return true
	case lexer.TokenPlusAssign, lexer.TokenMinusAssign, lexer.TokenStarAssign, lexer.TokenSlashAssign,
		lexer.TokenPercentAssign, lexer.TokenShiftLeftAssign, lexer.TokenShiftRightAssign,
		lexer.TokenAmpAssign, lexer.TokenCaretAssign, lexer.TokenPipeAssign:
		p.addDiagnostic(unsupportedError(tok, "compound assignment operators"))
		return true
	default:
		return false
	}
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
		case lexer.TokenEOF, lexer.TokenInt, lexer.TokenChar, lexer.TokenVoid:
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

func isTypeSpecifierToken(tt lexer.TokenType) bool {
	switch tt {
	case lexer.TokenInt, lexer.TokenChar, lexer.TokenVoid:
		return true
	default:
		return false
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

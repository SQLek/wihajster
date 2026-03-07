package parser

import (
	"errors"
	"strings"
	"testing"

	"github.com/SQLek/wihajster/internal/lexer"
)

type fakeTokenSource struct {
	tokens []lexer.Token
	idx    int
	failAt int
	err    error
}

func (s *fakeTokenSource) Peek() (lexer.Token, error) {
	if s.failAt >= 0 && s.idx >= s.failAt {
		return lexer.Token{}, s.err
	}
	if s.idx >= len(s.tokens) {
		return lexer.Token{Type: lexer.TokenEOF}, nil
	}
	return s.tokens[s.idx], nil
}

func (s *fakeTokenSource) Next() (lexer.Token, error) {
	tok, err := s.Peek()
	if err != nil {
		return lexer.Token{}, err
	}
	if s.idx < len(s.tokens) {
		s.idx++
	}
	return tok, nil
}

func tok(tt lexer.TokenType, raw string, line, col int) lexer.Token {
	return lexer.Token{Type: tt, Raw: []byte(raw), Line: line, Column: col}
}

func TestParseExpressionStatement_PrecedenceUnit(t *testing.T) {
	src := &fakeTokenSource{tokens: []lexer.Token{
		tok(lexer.TokenIntegerConstant, "2", 1, 1),
		tok(lexer.TokenPlus, "+", 1, 3),
		tok(lexer.TokenIntegerConstant, "2", 1, 5),
		tok(lexer.TokenStar, "*", 1, 7),
		tok(lexer.TokenIntegerConstant, "2", 1, 9),
		tok(lexer.TokenSemicolon, ";", 1, 10),
		tok(lexer.TokenEOF, "", 1, 11),
	}, failAt: -1}

	p := New(src)
	stmt, ok := p.parseExpressionStatement()
	if !ok {
		t.Fatalf("expected expression statement parse success, diagnostics=%v", p.diagnostics)
	}

	es, ok := stmt.(ExpressionStatement)
	if !ok {
		t.Fatalf("expected ExpressionStatement, got %T", stmt)
	}
	add, ok := es.Expression.(BinaryExpression)
	if !ok || add.Op != lexer.TokenPlus {
		t.Fatalf("expected top-level addition, got %#v", es.Expression)
	}
	mul, ok := add.RHS.(BinaryExpression)
	if !ok || mul.Op != lexer.TokenStar {
		t.Fatalf("expected multiplication on RHS, got %#v", add.RHS)
	}
	if len(p.diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %d", len(p.diagnostics))
	}
}

func TestParseBlockStatement_RecoversAtSemicolonAndKeepsSibling(t *testing.T) {
	src := &fakeTokenSource{tokens: []lexer.Token{
		tok(lexer.TokenLBrace, "{", 1, 1),
		tok(lexer.TokenIntegerConstant, "1", 2, 2),
		tok(lexer.TokenPlus, "+", 2, 4),
		tok(lexer.TokenSemicolon, ";", 2, 6),
		tok(lexer.TokenReturn, "return", 3, 2),
		tok(lexer.TokenIntegerConstant, "0", 3, 9),
		tok(lexer.TokenSemicolon, ";", 3, 10),
		tok(lexer.TokenRBrace, "}", 4, 1),
		tok(lexer.TokenEOF, "", 4, 2),
	}, failAt: -1}

	p := New(src)
	block, ok := p.parseBlockStatement()
	if !ok {
		t.Fatalf("expected block parse success with recovery, diagnostics=%v", p.diagnostics)
	}
	if len(p.diagnostics) == 0 {
		t.Fatalf("expected at least one diagnostic from broken expression statement")
	}
	if len(block.Statements) != 1 {
		t.Fatalf("expected 1 recovered sibling statement, got %d", len(block.Statements))
	}
	if _, ok := block.Statements[0].(ReturnStatement); !ok {
		t.Fatalf("expected recovered statement to be return, got %T", block.Statements[0])
	}
}

func TestParseTranslationUnit_FatalLexerErrorDominatesMessage(t *testing.T) {
	lexErr := errors.New("read failed")
	src := &fakeTokenSource{
		tokens: []lexer.Token{
			tok(lexer.TokenInt, "int", 1, 1),
			tok(lexer.TokenIdentifier, "main", 1, 5),
			tok(lexer.TokenLParen, "(", 1, 9),
			tok(lexer.TokenRParen, ")", 1, 10),
			tok(lexer.TokenLBrace, "{", 1, 12),
		},
		failAt: 5,
		err:    lexErr,
	}

	_, err := Parse(src)
	if err == nil {
		t.Fatalf("expected parse error")
	}
	pErrs, ok := err.(*ParseErrors)
	if !ok {
		t.Fatalf("expected *ParseErrors, got %T", err)
	}
	if pErrs.FatalLexer == nil {
		t.Fatalf("expected fatal lexer error")
	}
	if !strings.Contains(err.Error(), "lexer error") {
		t.Fatalf("expected top-level error to mention lexer error, got %q", err.Error())
	}
	if strings.Contains(err.Error(), "unexpected end of file") {
		t.Fatalf("expected lexer error to dominate over parser EOF diagnostics, got %q", err.Error())
	}
}

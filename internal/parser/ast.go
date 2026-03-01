package parser

import "github.com/SQLek/wihajster/internal/lexer"

type TypeSpecifier int

const (
	TypeSpecifierInt TypeSpecifier = iota
	TypeSpecifierVoid
)

type TranslationUnit struct {
	Functions []FunctionDefinition
}

type FunctionDefinition struct {
	ReturnType TypeSpecifier
	Name       string
	Body       BlockStatement
	Token      lexer.Token
}

type Statement interface {
	statementNode()
}

type BlockStatement struct {
	Token      lexer.Token
	Statements []Statement
}

func (BlockStatement) statementNode() {}

type ReturnStatement struct {
	Token      lexer.Token
	Expression Expression
}

func (ReturnStatement) statementNode() {}

type ExpressionStatement struct {
	Token      lexer.Token
	Expression Expression
}

func (ExpressionStatement) statementNode() {}

type IfStatement struct {
	Token lexer.Token
	Cond  Expression
	Then  Statement
	Else  Statement
}

func (IfStatement) statementNode() {}

type WhileStatement struct {
	Token lexer.Token
	Cond  Expression
	Body  Statement
}

func (WhileStatement) statementNode() {}

type Expression interface {
	expressionNode()
}

type IdentifierExpression struct {
	Token lexer.Token
	Name  string
}

func (IdentifierExpression) expressionNode() {}

type IntegerLiteralExpression struct {
	Token lexer.Token
	Raw   string
}

func (IntegerLiteralExpression) expressionNode() {}

type UnaryExpression struct {
	Token   lexer.Token
	Op      lexer.TokenType
	Operand Expression
}

func (UnaryExpression) expressionNode() {}

type BinaryExpression struct {
	Token lexer.Token
	Op    lexer.TokenType
	LHS   Expression
	RHS   Expression
}

func (BinaryExpression) expressionNode() {}

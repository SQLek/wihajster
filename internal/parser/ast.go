package parser

import "github.com/SQLek/wihajster/internal/lexer"

type TypeSpecifier int

const (
	TypeSpecifierInt TypeSpecifier = iota
	TypeSpecifierChar
	TypeSpecifierVoid
)

type TypeName struct {
	Token        lexer.Token
	Specifier    TypeSpecifier
	PointerDepth int
}

type TranslationUnit struct {
	Functions    []FunctionDefinition
	Declarations []Declaration
	Prototypes   []FunctionPrototype
}

type FunctionParameter struct {
	Token lexer.Token
	Type  TypeName
	Name  string
}

type FunctionDefinition struct {
	ReturnType TypeName
	Name       string
	Parameters []FunctionParameter
	Body       BlockStatement
	Token      lexer.Token
}

type FunctionPrototype struct {
	Token      lexer.Token
	ReturnType TypeName
	Name       string
	Parameters []FunctionParameter
}

type Declaration struct {
	Token       lexer.Token
	Type        TypeName
	Name        string
	Initializer Expression
}

type Statement interface {
	statementNode()
}

type BlockStatement struct {
	Token      lexer.Token
	Statements []Statement
}

func (BlockStatement) statementNode() {}

type DeclarationStatement struct {
	Token       lexer.Token
	Declaration Declaration
}

func (DeclarationStatement) statementNode() {}

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

type ForStatement struct {
	Token lexer.Token
	Init  Statement
	Cond  Expression
	Post  Expression
	Body  Statement
}

func (ForStatement) statementNode() {}

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

type CharacterLiteralExpression struct {
	Token lexer.Token
	Raw   string
}

func (CharacterLiteralExpression) expressionNode() {}

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

type AssignmentExpression struct {
	Token lexer.Token
	LHS   Expression
	RHS   Expression
}

func (AssignmentExpression) expressionNode() {}

type CallExpression struct {
	Token  lexer.Token
	Callee Expression
	Args   []Expression
}

func (CallExpression) expressionNode() {}

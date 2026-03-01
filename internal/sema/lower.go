package sema

import (
	"strconv"

	"github.com/SQLek/wihajster/internal/lexer"
	"github.com/SQLek/wihajster/internal/parser"
	"github.com/SQLek/wihajster/internal/tac"
)

type lowerer struct {
	fn          *tac.Function
	nextLabelID int
}

func Lower(tu *parser.TranslationUnit) (tac.Module, error) {
	mod := tac.Module{}
	for _, pfn := range tu.Functions {
		fn, err := lowerFunction(pfn)
		if err != nil {
			return tac.Module{}, err
		}
		mod.Functions = append(mod.Functions, fn)
	}
	return mod, nil
}

func lowerFunction(pfn parser.FunctionDefinition) (tac.Function, error) {
	fn := tac.Function{
		Name:       "@" + pfn.Name,
		ReturnType: lowerType(pfn.ReturnType),
	}
	if fn.ReturnType == "" {
		return tac.Function{}, unsupportedError(pfn.Token, "function return type")
	}

	l := &lowerer{fn: &fn}
	reachable, err := l.lowerStatement(pfn.Body)
	if err != nil {
		return tac.Function{}, err
	}
	if reachable {
		if pfn.ReturnType == parser.TypeSpecifierVoid {
			fn.AddRet("")
		} else {
			return tac.Function{}, newError(pfn.Token, "function %s may reach end without return", pfn.Name)
		}
	}

	return fn, nil
}

func lowerType(t parser.TypeSpecifier) string {
	switch t {
	case parser.TypeSpecifierInt:
		return "i32"
	case parser.TypeSpecifierVoid:
		return "void"
	default:
		return ""
	}
}

func (l *lowerer) lowerStatement(stmt parser.Statement) (bool, error) {
	switch s := stmt.(type) {
	case parser.BlockStatement:
		reachable := true
		for _, nested := range s.Statements {
			if !reachable {
				break
			}
			nextReachable, err := l.lowerStatement(nested)
			if err != nil {
				return false, err
			}
			reachable = nextReachable
		}
		return reachable, nil
	case parser.ExpressionStatement:
		if s.Expression == nil {
			return true, nil
		}
		_, err := l.lowerExpr(s.Expression)
		return true, err
	case parser.ReturnStatement:
		if s.Expression == nil {
			l.fn.AddRet("")
			return false, nil
		}
		val, err := l.lowerExpr(s.Expression)
		if err != nil {
			return false, err
		}
		l.fn.AddRet(val)
		return false, nil
	case parser.IfStatement:
		return l.lowerIfStatement(s)
	case parser.WhileStatement:
		return l.lowerWhileStatement(s)
	default:
		return false, unsupportedError(lexer.Token{}, "statement kind")
	}
}

func (l *lowerer) lowerIfStatement(s parser.IfStatement) (bool, error) {
	cond, err := l.lowerExpr(s.Cond)
	if err != nil {
		return false, err
	}

	thenLabel := l.newLabel()
	endLabel := l.newLabel()
	if s.Else == nil {
		l.fn.AddBr(cond, thenLabel, endLabel)
		l.fn.AddLabel(thenLabel)
		thenReachable, err := l.lowerStatement(s.Then)
		if err != nil {
			return false, err
		}
		if thenReachable {
			l.fn.AddJmp(endLabel)
		}
		l.fn.AddLabel(endLabel)
		return true, nil
	}

	elseLabel := l.newLabel()
	l.fn.AddBr(cond, thenLabel, elseLabel)

	l.fn.AddLabel(thenLabel)
	thenReachable, err := l.lowerStatement(s.Then)
	if err != nil {
		return false, err
	}
	if thenReachable {
		l.fn.AddJmp(endLabel)
	}

	l.fn.AddLabel(elseLabel)
	elseReachable, err := l.lowerStatement(s.Else)
	if err != nil {
		return false, err
	}
	if elseReachable {
		l.fn.AddJmp(endLabel)
	}

	if thenReachable || elseReachable {
		l.fn.AddLabel(endLabel)
		return true, nil
	}
	return false, nil
}

func (l *lowerer) lowerWhileStatement(s parser.WhileStatement) (bool, error) {
	condLabel := l.newLabel()
	bodyLabel := l.newLabel()
	endLabel := l.newLabel()

	l.fn.AddJmp(condLabel)
	l.fn.AddLabel(condLabel)
	cond, err := l.lowerExpr(s.Cond)
	if err != nil {
		return false, err
	}
	l.fn.AddBr(cond, bodyLabel, endLabel)

	l.fn.AddLabel(bodyLabel)
	bodyReachable, err := l.lowerStatement(s.Body)
	if err != nil {
		return false, err
	}
	if bodyReachable {
		l.fn.AddJmp(condLabel)
	}

	l.fn.AddLabel(endLabel)
	return true, nil
}

func (l *lowerer) lowerExpr(expr parser.Expression) (string, error) {
	switch e := expr.(type) {
	case parser.IntegerLiteralExpression:
		if _, err := strconv.ParseInt(e.Raw, 10, 32); err != nil {
			return "", newError(e.Token, "invalid integer literal %q", e.Raw)
		}
		return l.fn.AddInstruction("const.i32", e.Raw), nil
	case parser.IdentifierExpression:
		return "", unsupportedError(e.Token, "identifiers without declarations")
	case parser.UnaryExpression:
		operand, err := l.lowerExpr(e.Operand)
		if err != nil {
			return "", err
		}
		switch e.Op {
		case lexer.TokenPlus:
			return operand, nil
		case lexer.TokenMinus:
			return l.fn.AddInstruction("neg", operand), nil
		case lexer.TokenBang:
			return l.fn.AddInstruction("logic_not", operand), nil
		case lexer.TokenTilde:
			return l.fn.AddInstruction("not", operand), nil
		default:
			return "", unsupportedError(e.Token, "unary operator")
		}
	case parser.BinaryExpression:
		lhs, err := l.lowerExpr(e.LHS)
		if err != nil {
			return "", err
		}
		rhs, err := l.lowerExpr(e.RHS)
		if err != nil {
			return "", err
		}
		opcode := binaryOpcode(e.Op)
		if opcode == "" {
			return "", unsupportedError(e.Token, "binary operator")
		}
		if e.Op == lexer.TokenAndAnd || e.Op == lexer.TokenOrOr {
			lhs = l.fn.AddInstruction("ne", lhs, "0")
			rhs = l.fn.AddInstruction("ne", rhs, "0")
		}
		return l.fn.AddInstruction(opcode, lhs, rhs), nil
	default:
		return "", unsupportedError(lexer.Token{}, "expression kind")
	}
}

func binaryOpcode(op lexer.TokenType) string {
	switch op {
	case lexer.TokenPlus:
		return "add"
	case lexer.TokenMinus:
		return "sub"
	case lexer.TokenStar:
		return "mul"
	case lexer.TokenSlash:
		return "div_s"
	case lexer.TokenPercent:
		return "mod_s"
	case lexer.TokenAmp:
		return "and"
	case lexer.TokenPipe:
		return "or"
	case lexer.TokenCaret:
		return "xor"
	case lexer.TokenShiftLeft:
		return "shl"
	case lexer.TokenShiftRight:
		return "shr_s"
	case lexer.TokenEq:
		return "eq"
	case lexer.TokenNe:
		return "ne"
	case lexer.TokenLt:
		return "lt_s"
	case lexer.TokenLe:
		return "le_s"
	case lexer.TokenGt:
		return "gt_s"
	case lexer.TokenGe:
		return "ge_s"
	case lexer.TokenAndAnd:
		return "and"
	case lexer.TokenOrOr:
		return "or"
	default:
		return ""
	}
}

func (l *lowerer) newLabel() string {
	label := ".L" + strconv.Itoa(l.nextLabelID)
	l.nextLabelID++
	return label
}

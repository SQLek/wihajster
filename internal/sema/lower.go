package sema

import (
	"strconv"

	"github.com/SQLek/wihajster/internal/lexer"
	"github.com/SQLek/wihajster/internal/parser"
	"github.com/SQLek/wihajster/internal/tac"
)

type functionSignature struct {
	ReturnType string
	Params     []string
}

type variableSymbol struct {
	Type     string
	Slot     string
	IsGlobal bool
}

type lowerer struct {
	fn          *tac.Function
	nextLabelID int

	scopes  []map[string]variableSymbol
	globals map[string]variableSymbol
}

func Lower(tu *parser.TranslationUnit) (tac.Module, error) {
	globals := map[string]variableSymbol{}
	prototypes := map[string]functionSignature{}
	definitions := map[string]functionSignature{}

	for _, decl := range tu.Declarations {
		typ, err := lowerObjectType(decl.Token, decl.Type)
		if err != nil {
			return tac.Module{}, err
		}
		if _, exists := globals[decl.Name]; exists {
			return tac.Module{}, newError(decl.Token, "global %s redeclared", decl.Name)
		}
		if _, exists := prototypes[decl.Name]; exists {
			return tac.Module{}, newError(decl.Token, "name %s already declared as function", decl.Name)
		}
		if _, exists := definitions[decl.Name]; exists {
			return tac.Module{}, newError(decl.Token, "name %s already declared as function", decl.Name)
		}
		globals[decl.Name] = variableSymbol{Type: typ, IsGlobal: true}
	}

	for _, proto := range tu.Prototypes {
		sig, err := signatureForFunction(proto.Token, proto.ReturnType, proto.Parameters)
		if err != nil {
			return tac.Module{}, err
		}
		if _, exists := globals[proto.Name]; exists {
			return tac.Module{}, newError(proto.Token, "name %s already declared as global", proto.Name)
		}
		if prev, exists := prototypes[proto.Name]; exists {
			if !sameSignature(prev, sig) {
				return tac.Module{}, newError(proto.Token, "conflicting prototype for function %s", proto.Name)
			}
			continue
		}
		if def, exists := definitions[proto.Name]; exists && !sameSignature(def, sig) {
			return tac.Module{}, newError(proto.Token, "conflicting prototype for function %s", proto.Name)
		}
		prototypes[proto.Name] = sig
	}

	for _, pfn := range tu.Functions {
		sig, err := signatureForFunction(pfn.Token, pfn.ReturnType, pfn.Parameters)
		if err != nil {
			return tac.Module{}, err
		}
		if _, exists := globals[pfn.Name]; exists {
			return tac.Module{}, newError(pfn.Token, "name %s already declared as global", pfn.Name)
		}
		if prev, exists := definitions[pfn.Name]; exists {
			if sameSignature(prev, sig) {
				return tac.Module{}, newError(pfn.Token, "function %s defined multiple times", pfn.Name)
			}
			return tac.Module{}, newError(pfn.Token, "conflicting definition for function %s", pfn.Name)
		}
		if proto, exists := prototypes[pfn.Name]; exists && !sameSignature(proto, sig) {
			return tac.Module{}, newError(pfn.Token, "function definition does not match prototype for %s", pfn.Name)
		}
		definitions[pfn.Name] = sig
	}

	mod := tac.Module{}
	for _, pfn := range tu.Functions {
		fn, err := lowerFunction(pfn, globals)
		if err != nil {
			return tac.Module{}, err
		}
		mod.Functions = append(mod.Functions, fn)
	}
	return mod, nil
}

func lowerFunction(pfn parser.FunctionDefinition, globals map[string]variableSymbol) (tac.Function, error) {
	retType := lowerType(pfn.ReturnType)
	if retType == "" {
		return tac.Function{}, unsupportedError(pfn.Token, "function return type")
	}

	fn := tac.Function{Name: "@" + pfn.Name, ReturnType: retType}
	l := &lowerer{fn: &fn, globals: globals}
	l.pushScope()
	defer l.popScope()

	for _, param := range pfn.Parameters {
		paramType, err := lowerObjectType(param.Token, param.Type)
		if err != nil {
			return tac.Function{}, err
		}
		if err := l.declareLocal(param.Token, param.Name, paramType, "%"+param.Name); err != nil {
			return tac.Function{}, err
		}
		fn.Parameters = append(fn.Parameters, tac.Parameter{Name: "%" + param.Name, Type: paramType})

		slot := fn.AddInstruction("alloca", paramType)
		l.setLocalSlot(param.Name, slot)
		fn.AddVoidInstruction("store", slot, "%"+param.Name)
	}

	reachable, err := l.lowerBlockStatements(pfn.Body.Statements)
	if err != nil {
		return tac.Function{}, err
	}
	if reachable {
		if pfn.ReturnType.Specifier == parser.TypeSpecifierVoid {
			fn.AddRet("")
		} else {
			return tac.Function{}, newError(pfn.Token, "function %s may reach end without return", pfn.Name)
		}
	}

	return fn, nil
}

func signatureForFunction(tok lexer.Token, ret parser.TypeName, params []parser.FunctionParameter) (functionSignature, error) {
	retType := lowerType(ret)
	if retType == "" {
		return functionSignature{}, unsupportedError(tok, "function return type")
	}

	out := functionSignature{ReturnType: retType, Params: make([]string, 0, len(params))}
	seen := map[string]struct{}{}
	for _, param := range params {
		paramType, err := lowerObjectType(param.Token, param.Type)
		if err != nil {
			return functionSignature{}, err
		}
		if _, exists := seen[param.Name]; exists {
			return functionSignature{}, newError(param.Token, "parameter %s redeclared", param.Name)
		}
		seen[param.Name] = struct{}{}
		out.Params = append(out.Params, paramType)
	}
	return out, nil
}

func sameSignature(a, b functionSignature) bool {
	if a.ReturnType != b.ReturnType || len(a.Params) != len(b.Params) {
		return false
	}
	for i := range a.Params {
		if a.Params[i] != b.Params[i] {
			return false
		}
	}
	return true
}

func lowerType(t parser.TypeName) string {
	if t.PointerDepth > 0 {
		return "ptr"
	}
	switch t.Specifier {
	case parser.TypeSpecifierInt:
		return "i32"
	case parser.TypeSpecifierChar:
		return "i32"
	case parser.TypeSpecifierVoid:
		return "void"
	default:
		return ""
	}
}

func lowerObjectType(tok lexer.Token, t parser.TypeName) (string, error) {
	typ := lowerType(t)
	if typ == "" {
		return "", unsupportedError(tok, "declaration type")
	}
	if typ == "void" {
		return "", unsupportedError(tok, "void objects")
	}
	return typ, nil
}

func (l *lowerer) pushScope() {
	l.scopes = append(l.scopes, map[string]variableSymbol{})
}

func (l *lowerer) popScope() {
	l.scopes = l.scopes[:len(l.scopes)-1]
}

func (l *lowerer) currentScope() map[string]variableSymbol {
	return l.scopes[len(l.scopes)-1]
}

func (l *lowerer) setLocalSlot(name, slot string) {
	scope := l.currentScope()
	sym := scope[name]
	sym.Slot = slot
	scope[name] = sym
}

func (l *lowerer) declareLocal(tok lexer.Token, name, typ, slot string) error {
	scope := l.currentScope()
	if _, exists := scope[name]; exists {
		return newError(tok, "identifier %s redeclared in this scope", name)
	}
	scope[name] = variableSymbol{Type: typ, Slot: slot}
	return nil
}

func (l *lowerer) resolveVariable(name string) (variableSymbol, bool) {
	for i := len(l.scopes) - 1; i >= 0; i-- {
		if sym, ok := l.scopes[i][name]; ok {
			return sym, true
		}
	}
	sym, ok := l.globals[name]
	return sym, ok
}

func (l *lowerer) lowerBlockStatements(stmts []parser.Statement) (bool, error) {
	reachable := true
	for _, nested := range stmts {
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
}

func (l *lowerer) lowerStatement(stmt parser.Statement) (bool, error) {
	switch s := stmt.(type) {
	case parser.BlockStatement:
		l.pushScope()
		reachable, err := l.lowerBlockStatements(s.Statements)
		l.popScope()
		return reachable, err
	case parser.DeclarationStatement:
		return true, l.lowerLocalDeclaration(s.Declaration)
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
	case parser.ForStatement:
		return l.lowerForStatement(s)
	default:
		return false, unsupportedError(lexer.Token{}, "statement kind")
	}
}

func (l *lowerer) lowerLocalDeclaration(decl parser.Declaration) error {
	typ, err := lowerObjectType(decl.Token, decl.Type)
	if err != nil {
		return err
	}
	if err := l.declareLocal(decl.Token, decl.Name, typ, ""); err != nil {
		return err
	}
	slot := l.fn.AddInstruction("alloca", typ)
	l.setLocalSlot(decl.Name, slot)
	if decl.Initializer == nil {
		return nil
	}
	value, err := l.lowerExpr(decl.Initializer)
	if err != nil {
		return err
	}
	l.fn.AddVoidInstruction("store", slot, value)
	return nil
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

func (l *lowerer) lowerForStatement(s parser.ForStatement) (bool, error) {
	l.pushScope()
	defer l.popScope()

	if s.Init != nil {
		reachable, err := l.lowerStatement(s.Init)
		if err != nil {
			return false, err
		}
		if !reachable {
			return false, nil
		}
	}

	condLabel := l.newLabel()
	bodyLabel := l.newLabel()
	postLabel := l.newLabel()
	endLabel := l.newLabel()

	l.fn.AddJmp(condLabel)
	l.fn.AddLabel(condLabel)
	if s.Cond != nil {
		cond, err := l.lowerExpr(s.Cond)
		if err != nil {
			return false, err
		}
		l.fn.AddBr(cond, bodyLabel, endLabel)
	} else {
		l.fn.AddJmp(bodyLabel)
	}

	l.fn.AddLabel(bodyLabel)
	bodyReachable, err := l.lowerStatement(s.Body)
	if err != nil {
		return false, err
	}
	if bodyReachable {
		l.fn.AddJmp(postLabel)
	}

	l.fn.AddLabel(postLabel)
	if bodyReachable && s.Post != nil {
		if _, err := l.lowerExpr(s.Post); err != nil {
			return false, err
		}
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
	case parser.CharacterLiteralExpression:
		return "", unsupportedError(e.Token, "character literals")
	case parser.IdentifierExpression:
		sym, ok := l.resolveVariable(e.Name)
		if !ok {
			return "", newError(e.Token, "use of undeclared identifier %s", e.Name)
		}
		if sym.IsGlobal {
			return "", newError(e.Token, "not yet supported in M1 TAC lowering: global variable access")
		}
		return l.fn.AddInstruction("load", sym.Slot), nil
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
	case parser.AssignmentExpression:
		ident, ok := e.LHS.(parser.IdentifierExpression)
		if !ok {
			return "", unsupportedError(e.Token, "assignment target")
		}
		sym, ok := l.resolveVariable(ident.Name)
		if !ok {
			return "", newError(e.Token, "use of undeclared identifier %s", ident.Name)
		}
		if sym.IsGlobal {
			return "", newError(e.Token, "not yet supported in M1 TAC lowering: global variable access")
		}
		rhs, err := l.lowerExpr(e.RHS)
		if err != nil {
			return "", err
		}
		l.fn.AddVoidInstruction("store", sym.Slot, rhs)
		return rhs, nil
	case parser.CallExpression:
		return "", unsupportedError(e.Token, "function calls")
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

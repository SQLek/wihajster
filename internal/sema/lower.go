package sema

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/SQLek/wihajster/internal/lexer"
	"github.com/SQLek/wihajster/internal/parser"
	"github.com/SQLek/wihajster/internal/tac"
)

type functionSignature struct {
	ReturnType string
	Params     []string
}

type variableSymbol struct {
	Type string
	Slot string
}

type typedValue struct {
	Value tac.Operand
	Type  string
}

type lowerer struct {
	fn          *tac.Function
	nextLabelID int

	returnType string
	functions  map[string]functionSignature
	scopes     []map[string]variableSymbol
}

func Lower(tu *parser.TranslationUnit) (tac.Module, error) {
	if len(tu.Declarations) > 0 {
		return tac.Module{}, unsupportedError(tu.Declarations[0].Token, "global declarations")
	}

	prototypes := map[string]functionSignature{}
	definitions := map[string]functionSignature{}

	for _, proto := range tu.Prototypes {
		sig, err := signatureForFunction(proto.Token, proto.ReturnType, proto.Parameters)
		if err != nil {
			return tac.Module{}, err
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

	functions := map[string]functionSignature{}
	for name, sig := range prototypes {
		functions[name] = sig
	}
	for name, sig := range definitions {
		functions[name] = sig
	}

	mod := tac.Module{}
	for _, pfn := range tu.Functions {
		fn, err := lowerFunction(pfn, functions)
		if err != nil {
			return tac.Module{}, err
		}
		mod.Functions = append(mod.Functions, fn)
	}
	return mod, nil
}

func lowerFunction(pfn parser.FunctionDefinition, functions map[string]functionSignature) (tac.Function, error) {
	retType := lowerType(pfn.ReturnType)
	if retType == "" {
		return tac.Function{}, unsupportedError(pfn.Token, "function return type")
	}

	fn := tac.Function{Name: "@" + pfn.Name, ReturnType: retType}
	l := &lowerer{fn: &fn, returnType: retType, functions: functions}
	l.pushScope()
	defer l.popScope()

	for _, param := range pfn.Parameters {
		paramType, err := lowerObjectType(param.Token, param.Type)
		if err != nil {
			return tac.Function{}, err
		}
		if err := l.declareLocal(param.Token, param.Name, paramType); err != nil {
			return tac.Function{}, err
		}
		fn.Parameters = append(fn.Parameters, tac.Parameter{Name: "%" + param.Name, Type: paramType})

		slot := fn.AddInstruction(tac.OpcodeAlloca, tac.Immediate(paramType))
		l.setLocalSlot(param.Name, slot.Text)
		fn.AddVoidInstruction(tac.OpcodeStore, slot, tac.Param("%"+param.Name))
	}

	reachable, err := l.lowerBlockStatements(pfn.Body.Statements)
	if err != nil {
		return tac.Function{}, err
	}
	if reachable {
		if pfn.ReturnType.Specifier == parser.TypeSpecifierVoid {
			fn.AddRet(tac.Operand{})
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
	base := ""
	switch t.Specifier {
	case parser.TypeSpecifierInt:
		base = "i32"
	case parser.TypeSpecifierChar:
		base = "i32"
	case parser.TypeSpecifierVoid:
		base = "void"
	default:
		return ""
	}
	for i := 0; i < t.PointerDepth; i++ {
		base += "*"
	}
	return base
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

func (l *lowerer) declareLocal(tok lexer.Token, name, typ string) error {
	scope := l.currentScope()
	if _, exists := scope[name]; exists {
		return newError(tok, "identifier %s redeclared in this scope", name)
	}
	scope[name] = variableSymbol{Type: typ}
	return nil
}

func (l *lowerer) resolveVariable(name string) (variableSymbol, bool) {
	for i := len(l.scopes) - 1; i >= 0; i-- {
		if sym, ok := l.scopes[i][name]; ok {
			return sym, true
		}
	}
	return variableSymbol{}, false
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
			if l.returnType != "void" {
				return false, newError(s.Token, "non-void function must return a value")
			}
			l.fn.AddRet(tac.Operand{})
			return false, nil
		}

		val, err := l.lowerExpr(s.Expression)
		if err != nil {
			return false, err
		}
		if l.returnType == "void" {
			return false, newError(s.Token, "void function must not return a value")
		}
		if val.Type != l.returnType {
			return false, newError(s.Token, "return type mismatch: expected %s, got %s", l.returnType, val.Type)
		}
		l.fn.AddRet(val.Value)
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
	if err := l.declareLocal(decl.Token, decl.Name, typ); err != nil {
		return err
	}
	slot := l.fn.AddInstruction(tac.OpcodeAlloca, tac.Immediate(typ))
	l.setLocalSlot(decl.Name, slot.Text)
	if decl.Initializer == nil {
		return nil
	}
	value, err := l.lowerExpr(decl.Initializer)
	if err != nil {
		return err
	}
	if value.Type != typ {
		return newError(decl.Token, "initializer type mismatch for %s: expected %s, got %s", decl.Name, typ, value.Type)
	}
	l.fn.AddVoidInstruction(tac.OpcodeStore, slot, value.Value)
	return nil
}

func (l *lowerer) lowerIfStatement(s parser.IfStatement) (bool, error) {
	cond, err := l.lowerExpr(s.Cond)
	if err != nil {
		return false, err
	}
	if cond.Type == "void" {
		return false, newError(s.Token, "if condition cannot have void type")
	}

	thenLabel := l.newLabel()
	endLabel := l.newLabel()
	if s.Else == nil {
		l.fn.AddBr(cond.Value, thenLabel, endLabel)
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
	l.fn.AddBr(cond.Value, thenLabel, elseLabel)

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
	if cond.Type == "void" {
		return false, newError(s.Token, "while condition cannot have void type")
	}
	l.fn.AddBr(cond.Value, bodyLabel, endLabel)

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
		if cond.Type == "void" {
			return false, newError(s.Token, "for condition cannot have void type")
		}
		l.fn.AddBr(cond.Value, bodyLabel, endLabel)
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

func (l *lowerer) lowerExpr(expr parser.Expression) (typedValue, error) {
	switch e := expr.(type) {
	case parser.IntegerLiteralExpression:
		if _, err := strconv.ParseInt(e.Raw, 10, 32); err != nil {
			return typedValue{}, newError(e.Token, "invalid integer literal %q", e.Raw)
		}
		return typedValue{Value: l.fn.AddInstruction(tac.OpcodeConstI32, tac.Immediate(e.Raw)), Type: "i32"}, nil
	case parser.CharacterLiteralExpression:
		value, err := decodeCharacterLiteral(e.Raw)
		if err != nil {
			return typedValue{}, newError(e.Token, "%s", err.Error())
		}
		return typedValue{Value: l.fn.AddInstruction(tac.OpcodeConstI32, tac.Immediate(strconv.FormatInt(int64(value), 10))), Type: "i32"}, nil
	case parser.IdentifierExpression:
		sym, ok := l.resolveVariable(e.Name)
		if !ok {
			return typedValue{}, newError(e.Token, "use of undeclared identifier %s", e.Name)
		}
		return typedValue{Value: l.fn.AddInstruction(tac.OpcodeLoad, tac.StackSlotPointer(sym.Slot)), Type: sym.Type}, nil
	case parser.UnaryExpression:
		switch e.Op {
		case lexer.TokenStar:
			ptr, err := l.lowerExpr(e.Operand)
			if err != nil {
				return typedValue{}, err
			}
			elemType, ok := pointeeType(ptr.Type)
			if !ok {
				return typedValue{}, newError(e.Token, "cannot dereference non-pointer type %s", ptr.Type)
			}
			if elemType == "void" {
				return typedValue{}, newError(e.Token, "cannot dereference void* without cast")
			}
			return typedValue{Value: l.loadAddress(ptr.Value), Type: elemType}, nil
		case lexer.TokenAmp:
			addr, elemType, err := l.lowerAddress(e.Operand)
			if err != nil {
				return typedValue{}, err
			}
			return typedValue{Value: addr, Type: pointerTo(elemType)}, nil
		}

		operand, err := l.lowerExpr(e.Operand)
		if err != nil {
			return typedValue{}, err
		}
		if operand.Type == "void" {
			return typedValue{}, newError(e.Token, "unary operator requires non-void operand")
		}
		if isPointerType(operand.Type) {
			return typedValue{}, newError(e.Token, "unary operator %v does not accept pointer operand", e.Op)
		}
		switch e.Op {
		case lexer.TokenPlus:
			return operand, nil
		case lexer.TokenMinus:
			return typedValue{Value: l.fn.AddInstruction(tac.OpcodeNeg, operand.Value), Type: operand.Type}, nil
		case lexer.TokenBang:
			return typedValue{Value: l.fn.AddInstruction(tac.OpcodeLogicNot, operand.Value), Type: "i32"}, nil
		case lexer.TokenTilde:
			return typedValue{Value: l.fn.AddInstruction(tac.OpcodeNot, operand.Value), Type: operand.Type}, nil
		default:
			return typedValue{}, unsupportedError(e.Token, "unary operator")
		}
	case parser.BinaryExpression:
		lhs, err := l.lowerExpr(e.LHS)
		if err != nil {
			return typedValue{}, err
		}
		rhs, err := l.lowerExpr(e.RHS)
		if err != nil {
			return typedValue{}, err
		}
		if lhs.Type == "void" || rhs.Type == "void" {
			return typedValue{}, newError(e.Token, "binary operator requires non-void operands")
		}
		opcode := binaryOpcode(e.Op)
		if opcode == tac.OpcodeInvalid {
			return typedValue{}, unsupportedError(e.Token, "binary operator")
		}
		if e.Op == lexer.TokenAndAnd || e.Op == lexer.TokenOrOr {
			lhsVal := l.fn.AddInstruction(tac.OpcodeNe, lhs.Value, tac.Immediate("0"))
			rhsVal := l.fn.AddInstruction(tac.OpcodeNe, rhs.Value, tac.Immediate("0"))
			return typedValue{Value: l.fn.AddInstruction(opcode, lhsVal, rhsVal), Type: "i32"}, nil
		}

		resultType := lhs.Type
		switch e.Op {
		case lexer.TokenEq, lexer.TokenNe, lexer.TokenLt, lexer.TokenLe, lexer.TokenGt, lexer.TokenGe:
			resultType = "i32"
		}
		return typedValue{Value: l.fn.AddInstruction(opcode, lhs.Value, rhs.Value), Type: resultType}, nil
	case parser.AssignmentExpression:
		addr, lhsType, err := l.lowerAddress(e.LHS)
		if err != nil {
			return typedValue{}, err
		}
		rhs, err := l.lowerExpr(e.RHS)
		if err != nil {
			return typedValue{}, err
		}
		if rhs.Type != lhsType {
			return typedValue{}, newError(e.Token, "assignment type mismatch: expected %s, got %s", lhsType, rhs.Type)
		}
		l.storeAddress(addr, rhs.Value)
		return rhs, nil
	case parser.CallExpression:
		callee, ok := e.Callee.(parser.IdentifierExpression)
		if !ok {
			return typedValue{}, unsupportedError(e.Token, "function call target")
		}
		sig, exists := l.functions[callee.Name]
		if !exists {
			return typedValue{}, newError(callee.Token, "call to undeclared function %s", callee.Name)
		}
		if len(e.Args) != len(sig.Params) {
			return typedValue{}, newError(e.Token, "function %s expects %d arguments, got %d", callee.Name, len(sig.Params), len(e.Args))
		}
		args := make([]tac.Operand, 0, len(e.Args))
		for i, argExpr := range e.Args {
			arg, err := l.lowerExpr(argExpr)
			if err != nil {
				return typedValue{}, err
			}
			expected := sig.Params[i]
			if arg.Type != expected {
				return typedValue{}, newError(e.Token, "argument %d to %s has type %s, expected %s", i+1, callee.Name, arg.Type, expected)
			}
			args = append(args, arg.Value)
		}
		calleeName := "@" + callee.Name
		if sig.ReturnType == "void" {
			l.fn.AddCallVoid(tac.FunctionSymbol(calleeName), args...)
			return typedValue{Type: "void"}, nil
		}
		return typedValue{Value: l.fn.AddCall(tac.FunctionSymbol(calleeName), args...), Type: sig.ReturnType}, nil
	default:
		return typedValue{}, unsupportedError(lexer.Token{}, "expression kind")
	}
}

func (l *lowerer) lowerAddress(expr parser.Expression) (tac.Operand, string, error) {
	switch e := expr.(type) {
	case parser.IdentifierExpression:
		sym, ok := l.resolveVariable(e.Name)
		if !ok {
			return tac.Operand{}, "", newError(e.Token, "use of undeclared identifier %s", e.Name)
		}
		return tac.StackSlotPointer(sym.Slot), sym.Type, nil
	case parser.UnaryExpression:
		if e.Op != lexer.TokenStar {
			break
		}
		ptr, err := l.lowerExpr(e.Operand)
		if err != nil {
			return tac.Operand{}, "", err
		}
		elemType, ok := pointeeType(ptr.Type)
		if !ok {
			return tac.Operand{}, "", newError(e.Token, "cannot dereference non-pointer type %s", ptr.Type)
		}
		if elemType == "void" {
			return tac.Operand{}, "", newError(e.Token, "cannot dereference void* without cast")
		}
		return ptr.Value, elemType, nil
	}
	return tac.Operand{}, "", unsupportedError(lexer.Token{}, "assignment target")
}

func isPointerType(typ string) bool {
	return strings.HasSuffix(typ, "*")
}

func pointeeType(typ string) (string, bool) {
	if !isPointerType(typ) {
		return "", false
	}
	return typ[:len(typ)-1], true
}

func pointerTo(typ string) string {
	return typ + "*"
}

func (l *lowerer) loadAddress(addr tac.Operand) tac.Operand {
	if addr.Kind == tac.OperandStackSlotPointer {
		return l.fn.AddInstruction(tac.OpcodeLoad, addr)
	}
	return l.fn.AddInstruction(tac.OpcodeLoadIndirect, addr)
}

func (l *lowerer) storeAddress(addr, value tac.Operand) {
	if addr.Kind == tac.OperandStackSlotPointer {
		l.fn.AddVoidInstruction(tac.OpcodeStore, addr, value)
		return
	}
	l.fn.AddVoidInstruction(tac.OpcodeStoreIndirect, addr, value)
}

func decodeCharacterLiteral(raw string) (int32, error) {
	if len(raw) < 3 || raw[0] != '\'' || raw[len(raw)-1] != '\'' {
		return 0, fmt.Errorf("invalid character literal %q", raw)
	}
	body := raw[1 : len(raw)-1]
	if len(body) == 0 {
		return 0, fmt.Errorf("invalid character literal %q", raw)
	}
	if body[0] != '\\' {
		if len(body) != 1 {
			return 0, fmt.Errorf("multi-character literals are unsupported: %q", raw)
		}
		return int32(body[0]), nil
	}
	if len(body) != 2 {
		return 0, fmt.Errorf("invalid escape in character literal %q", raw)
	}
	switch body[1] {
	case '\\':
		return int32('\\'), nil
	case '\'':
		return int32('\''), nil
	case 'n':
		return int32('\n'), nil
	case 't':
		return int32('\t'), nil
	case 'r':
		return int32('\r'), nil
	case '0':
		return int32(0), nil
	default:
		return 0, fmt.Errorf("unsupported escape in character literal %q", raw)
	}
}

func binaryOpcode(op lexer.TokenType) tac.Opcode {
	switch op {
	case lexer.TokenPlus:
		return tac.OpcodeAdd
	case lexer.TokenMinus:
		return tac.OpcodeSub
	case lexer.TokenStar:
		return tac.OpcodeMul
	case lexer.TokenSlash:
		return tac.OpcodeDivS
	case lexer.TokenPercent:
		return tac.OpcodeModS
	case lexer.TokenAmp:
		return tac.OpcodeAnd
	case lexer.TokenPipe:
		return tac.OpcodeOr
	case lexer.TokenCaret:
		return tac.OpcodeXor
	case lexer.TokenShiftLeft:
		return tac.OpcodeShl
	case lexer.TokenShiftRight:
		return tac.OpcodeShrS
	case lexer.TokenEq:
		return tac.OpcodeEq
	case lexer.TokenNe:
		return tac.OpcodeNe
	case lexer.TokenLt:
		return tac.OpcodeLtS
	case lexer.TokenLe:
		return tac.OpcodeLeS
	case lexer.TokenGt:
		return tac.OpcodeGtS
	case lexer.TokenGe:
		return tac.OpcodeGeS
	case lexer.TokenAndAnd:
		return tac.OpcodeAnd
	case lexer.TokenOrOr:
		return tac.OpcodeOr
	default:
		return tac.OpcodeInvalid
	}
}

func (l *lowerer) newLabel() string {
	label := ".L" + strconv.Itoa(l.nextLabelID)
	l.nextLabelID++
	return label
}

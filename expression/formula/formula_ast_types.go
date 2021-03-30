package formula

import (
	e "github.com/google/xtoproto/expression"

	irpb "github.com/google/xtoproto/proto/expression/formulair"
)

type expr interface {
	isExpr()
	SourceContext() *irpb.AST_SourceContext
	astProto() *irpb.AST_Expression
}

// commonExpr contains information that's common across all expression subtypes.
type commonExpr struct {
	srcContext *irpb.AST_SourceContext
}

func (ce *commonExpr) isExpr() {}

func (ce *commonExpr) SourceContext() *irpb.AST_SourceContext {
	return ce.srcContext
}

func (ce *commonExpr) baseProto() *irpb.AST_Expression {
	return &irpb.AST_Expression{
		SourceContext: ce.SourceContext(),
	}
}

type voidExpr struct {
	commonExpr
}

func (ex *voidExpr) astProto() *irpb.AST_Expression {
	out := ex.baseProto()
	return out
}

type constantExpr struct {
	commonExpr
	value *e.Expression
}

func (ex *constantExpr) astProto() *irpb.AST_Expression {
	out := ex.baseProto()
	out.Value = &irpb.AST_Expression_Constant{
		Constant: &irpb.AST_Constant{
			Value: ex.value.Proto(),
		},
	}
	return out
}

type funcallExpr struct {
	commonExpr
	function expr
	args     []expr
}

func (ex *funcallExpr) astProto() *irpb.AST_Expression {
	var argProtos []*irpb.AST_Expression
	for _, arg := range ex.args {
		argProtos = append(argProtos, arg.astProto())
	}

	out := ex.baseProto()
	out.Value = &irpb.AST_Expression_Funcall{
		Funcall: &irpb.AST_FunctionCall{
			Function:       ex.function.astProto(),
			PositionalArgs: argProtos,
		},
	}
	return out
}

type functionExpr struct {
	commonExpr
}

func (ex *functionExpr) astProto() *irpb.AST_Expression {
	out := ex.baseProto()
	return out
}

type variableRefExpr struct {
	commonExpr
	sym *e.Symbol
}

func (ex *variableRefExpr) astProto() *irpb.AST_Expression {
	out := ex.baseProto()
	out.Value = &irpb.AST_Expression_Variable{
		Variable: &irpb.AST_VariableRef{
			Symbol: ex.sym.Proto(),
		},
	}
	return out
}

type ifElseExpr struct {
	commonExpr
	test, then, elseExpr expr
}

func (ex *ifElseExpr) astProto() *irpb.AST_Expression {
	var elseProto *irpb.AST_Expression
	if ex.elseExpr != nil {
		elseProto = ex.elseExpr.astProto()
	}
	out := ex.baseProto()
	out.Value = &irpb.AST_Expression_IfElse{
		IfElse: &irpb.AST_IfElse{
			Test:           ex.test.astProto(),
			ThenExpression: ex.then.astProto(),
			ElseExpression: elseProto,
		},
	}
	return out
}

type whileLoopExpr struct {
	commonExpr
}

func (ex *whileLoopExpr) astProto() *irpb.AST_Expression {
	out := ex.baseProto()
	return out
}

type letExpr struct {
	commonExpr
}

func (ex *letExpr) astProto() *irpb.AST_Expression {
	out := ex.baseProto()
	return out
}

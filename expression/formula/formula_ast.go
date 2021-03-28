package formula

import (
	"github.com/google/xtoproto/expression"

	irpb "github.com/google/xtoproto/proto/internal/formulair"
)

// abstract syntax tree for formulas

// An object that changes as the syntax tree is traversed.
type compilerContext struct {
	lexEnv *lexEnv
}

type CompiledExpression struct {
}

func (ce *CompiledExpression) String() string {
	panic("v interface{}")
}

func (ce *CompiledExpression) astProto() *irpb.AST_Expression {
	panic("v interface{}")
}

func Compile(exp *expression.Expression) (*CompiledExpression, error) {
	cctx := &compilerContext{
		lexEnv: defaultEvalContext().lexEnv,
	}
	return compile(cctx, exp)
}

func compile(cctx *compilerContext, exp *expression.Expression) (*CompiledExpression, error) {
}

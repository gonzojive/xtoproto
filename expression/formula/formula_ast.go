package formula

import (
	"fmt"

	"github.com/google/xtoproto/expression"
	"github.com/google/xtoproto/expression/expressions"
	"google.golang.org/protobuf/encoding/prototext"

	irpb "github.com/google/xtoproto/proto/expression/formulair"
)

// abstract syntax tree for formulas

// An object that changes as the syntax tree is traversed.
type compileContext struct {
	lexEnv *lexEnv
}

func (cctx *compileContext) errorf(format string, arg ...interface{}) error {
	return fmt.Errorf(format, arg...)
}

func (cctx *compileContext) commonExpr(form *expression.Expression) commonExpr {
	return commonExpr{
		srcContext: &irpb.AST_SourceContext{
			Context: form.Proto().GetSourceContext(),
		},
	}
}

type CompiledExpression struct {
	expr expr
}

func (ce *CompiledExpression) String() string {
	return prototext.Format(ce.astProto())
}

func (ce *CompiledExpression) astProto() *irpb.AST_Expression {
	return ce.expr.astProto()
}

func Compile(exp *expression.Expression) (*CompiledExpression, error) {
	cctx := &compileContext{
		lexEnv: defaultEvalContext().lexEnv,
	}
	ex, err := compile(cctx, exp)
	if err != nil {
		return nil, err
	}
	return &CompiledExpression{ex}, nil
}

func compile(cctx *compileContext, exp *expression.Expression) (expr, error) {
	// First we check the type of the expression. Literals evaluate to
	// their read values.
	unevaled := exp.Value()
	switch v := unevaled.(type) {
	case int:
		return compileConst(cctx, exp)
	case int8:
		return compileConst(cctx, exp)
	case int16:
		return compileConst(cctx, exp)
	case int32:
		return compileConst(cctx, exp)
	case int64:
		return compileConst(cctx, exp)
	case uint:
		return compileConst(cctx, exp)
	case uint16:
		return compileConst(cctx, exp)
	case uint32:
		return compileConst(cctx, exp)
	case uint64:
		return compileConst(cctx, exp)
	case float32:
		return compileConst(cctx, exp)
	case float64:
		return compileConst(cctx, exp)
	case string:
		return compileConst(cctx, exp)
	case []byte:
		return compileConst(cctx, exp)
	case *expression.Symbol:
		return compiledVariableRef(cctx, exp, v), nil
	case *expression.List:
		if v.Len() == 0 {
			return nil, cctx.errorf("unsupported form: empty list")
		}
		vals := v.Slice()
		operator, ok := vals[0].Value().(*expression.Symbol)
		if !ok {
			return nil, cctx.errorf("first argument in an s-expression must be a symbol, got %v", vals[0])
		}

		boundValue := cctx.lexEnv.resolve(operator, irpb.Binding_FUNCTION)
		switch def := boundValue.(type) {
		case *specialFormDef:
			return def.compile(cctx, exp)
		case *fnDef:
			fn := compiledVariableRef(cctx, vals[0], operator)
			return compileFuncall(cctx, exp, fn, expression.NewList(vals[1:]))
		default:
			return nil, cctx.errorf("unsupported operator %s in form %s", operator.ExpressionString(), exp.ExpressionString())
		}
	default:
		return nil, cctx.errorf("unsupported form: %s", exp.ExpressionString())
	}
}

func compileFuncall(cctx *compileContext, form *expression.Expression, fn expr, args *expression.List) (expr, error) {
	var argExprs []expr
	for i, arg := range args.Slice() {
		compiledArg, err := compile(cctx, arg)
		if err != nil {
			return nil, cctx.errorf("error compiling funcall argument %d: %w", i, err)
		}
		argExprs = append(argExprs, compiledArg)
	}
	return &funcallExpr{
		commonExpr: cctx.commonExpr(form),
		function:   fn,
		args:       argExprs,
	}, nil
}

func compiledVariableRef(cctx *compileContext, form *expression.Expression, sym *expression.Symbol) expr {
	return &variableRefExpr{
		commonExpr: cctx.commonExpr(form),
		sym:        sym,
	}
}

func compileConst(cctx *compileContext, form *expression.Expression) (expr, error) {
	var constExpr *expression.Expression
	unevaled := form.Value()
	switch v := unevaled.(type) {
	case int:
		constExpr = expression.FromInt(int(v))
	case int8:
		constExpr = expression.FromInt(int(v))
	case int16:
		constExpr = expression.FromInt(int(v))
	case int32:
		constExpr = expression.FromInt(int(v))
	case int64:
		constExpr = expression.FromInt(int(v))
	case uint:
		constExpr = expression.FromInt(int(v))
	case uint16:
		constExpr = expression.FromInt(int(v))
	case uint32:
		constExpr = expression.FromInt(int(v))
	case uint64:
		constExpr = expression.FromInt(int(v))
	case string:
		constExpr = expression.FromString(v)
	case float32:
		constExpr = expression.FromFloat32(v)
	case float64:
		constExpr = expression.FromFloat64(v)
	default:
		return nil, cctx.errorf("unsupported type encountered while compiling constant: %v", v)
	}

	return &constantExpr{
		commonExpr: cctx.commonExpr(form),
		value:      constExpr,
	}, nil
}

func compileIfElse(cctx *compileContext, form *expression.Expression) (expr, error) {
	parsed := struct {
		Op   *expression.Symbol
		Test *expression.Expression
		Rest []*expression.Expression `sexpr:"&rest"`
	}{}

	if err := expressions.Bind(form, &parsed); err != nil {
		return nil, cctx.errorf("error parsing if/else: %w", err)
	}
	if len(parsed.Rest) == 0 {
		return nil, cctx.errorf("error parsing if/else: must have a THEN form")
	}
	if len(parsed.Rest) > 2 {
		return nil, cctx.errorf("error parsing if/else: must have only a THEN and ELSE form")
	}
	test, err := compile(cctx, parsed.Test)
	if err != nil {
		return nil, cctx.errorf("error parsing if/else TEST clause: %w", err)
	}
	then, err := compile(cctx, parsed.Rest[0])
	if err != nil {
		return nil, cctx.errorf("error parsing if/else THEN clause: %w", err)
	}
	var elseExpr expr
	if len(parsed.Rest) == 2 {
		x, err := compile(cctx, parsed.Rest[0])
		if err != nil {
			return nil, cctx.errorf("error parsing if/else ELSE clause: %w", err)
		}
		elseExpr = x
	}

	return &ifElseExpr{
		commonExpr: cctx.commonExpr(form),
		test:       test,
		then:       then,
		elseExpr:   elseExpr,
	}, nil
}

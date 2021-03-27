// Package formula is used to express simple expressions that can be compiled
// into multiple languages.
package formula

import (
	"fmt"

	"github.com/google/xtoproto/expression"
)

// EvalContext is passed to functions during evaluation.
type EvalContext struct {
	// Since we don't compile the expression ahead of
	lexEnv *lexEnv

	// Description of location in a source file relevant to the current
	// evaluation.
	sourceLocation string
}

func defaultEvalContext() *EvalContext {
	return &EvalContext{
		&lexEnv{
			fnDefs: builtinFunctions,
		},
		"",
	}
}

// String returns a summary of the
func (ectx *EvalContext) String() string {
	return "nil EvalContext"
}

// errorf returns an error with some contextual formatting based on the original
// location of the error.
func (ectx *EvalContext) errorf(format string, arg ...interface{}) error {
	return fmt.Errorf(format, arg...)
}

// Eval evaluates the given expression according to the semantics in the package
// description.
func Eval(exp *expression.Expression) (Value, error) {
	ectx := defaultEvalContext()
	// First we check the type of the expression. Literals evaluate to
	// their read values.
	unevaled := exp.Value()
	switch v := unevaled.(type) {
	case int, int8, int16, int32, int64, uint, uint16, uint32, uint64, string, float32, float64, []byte:
		return v, nil
	case *expression.List:
		if v.Len() == 0 {
			return nil, ectx.errorf("unsupported form: empty list")
		}
		vals := v.Slice()
		operator, ok := vals[0].Value().(*expression.Symbol)
		if !ok {
			return nil, ectx.errorf("first argument in an s-expression must be a symbol, got %v", vals[0])
		}
		fn := ectx.lexEnv.resolveFnDef(operator)
		if fn == nil {
			return nil, ectx.errorf("failed to resolve function %s", operator.ExpressionString())
		}
		return fn.impl(ectx, listValues(v)[1:])
	}
	return nil, fmt.Errorf("unsupported expression: %s", exp.ExpressionString())
}

// symKey is used when a symbol is needed in a map.
type symKey struct {
	Name      string
	Namespace expression.Namespace
}

func (k symKey) String() string             { return k.Symbol().String() }
func (k symKey) Symbol() *expression.Symbol { return expression.NewSymbol(k.Name, k.Namespace) }

type fnDef struct {
	name *expression.Symbol
	impl func(ectx *EvalContext, args []Value) (Value, error)
}

// Value is an evaluated value.
type Value interface{}

// lexEnv is used to represent the lexical environment. It contains
// function bindings to symbols, variable bindings, etc.
type lexEnv struct {
	fnDefs []*fnDef
}

func (le *lexEnv) copy() *lexEnv {
	out := &lexEnv{}
	out.fnDefs = append(out.fnDefs, le.fnDefs...)
	return out
}

func (le *lexEnv) resolveFnDef(name *expression.Symbol) *fnDef {
	name = normalizeSym(name)
	for _, fd := range le.fnDefs {
		if fd.name.Equals(name) {
			return fd
		}
	}
	return nil
}

func listValues(l *expression.List) []Value {
	var out []Value
	for i := 0; i < l.Len(); i++ {
		out = append(out, l.Nth(i).Value())
	}
	return out
}

func normalizeSym(s *expression.Symbol) *expression.Symbol {
	if s.Namespace() == "" {
		return expression.NewSymbol(s.Name(), namespace)
	}
	return s
}

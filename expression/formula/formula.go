// Package formula is used to express simple expressions that can be compiled
// into multiple languages.
package formula

import (
	"fmt"

	e "github.com/google/xtoproto/expression"
	irpb "github.com/google/xtoproto/proto/expression/formulair"
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
	out := &EvalContext{
		&lexEnv{},
		"",
	}

	for _, f := range builtinFunctions {
		out.lexEnv = out.lexEnv.withFunctionDef(f)
	}
	for _, sf := range defaultSpecialForms {
		out.lexEnv = out.lexEnv.withBinding(sf)
	}
	return out
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
func Eval(exp *e.Expression) (Value, error) {
	ectx := defaultEvalContext()
	// First we check the type of the expression. Literals evaluate to
	// their read values.
	unevaled := exp.Value()
	switch v := unevaled.(type) {
	case int, int8, int16, int32, int64, uint, uint16, uint32, uint64, string, float32, float64, []byte:
		return v, nil
	case *e.List:
		if v.Len() == 0 {
			return nil, ectx.errorf("unsupported form: empty list")
		}
		vals := v.Slice()
		operator, ok := vals[0].Value().(*e.Symbol)
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
	Namespace e.Namespace
}

func (k symKey) String() string    { return k.Symbol().String() }
func (k symKey) Symbol() *e.Symbol { return e.NewSymbol(k.Name, k.Namespace) }

type fnDef struct {
	name *e.Symbol
	impl func(ectx *EvalContext, args []Value) (Value, error)
}

func (def *fnDef) variableName() *e.Symbol {
	return def.name
}

func (def *fnDef) kind() irpb.Binding_Kind {
	return irpb.Binding_FUNCTION
}

type specialFormDef struct {
	name    *e.Symbol
	compile func(cctx *compileContext, wholeForm *e.Expression) (expr, error)
}

func (def *specialFormDef) variableName() *e.Symbol {
	return def.name
}

func (def *specialFormDef) kind() irpb.Binding_Kind {
	return irpb.Binding_FUNCTION
}

// Value is an evaluated value.
type Value interface{}

type lexEnvEntry interface {
	// The symbol bound to a value in a lexical environment.
	variableName() *e.Symbol

	// Function, variable, or some other namespace. Note that function
	// is used for macros and special forms, too.
	kind() irpb.Binding_Kind
}

// lexEnv is used to represent the lexical environment. It contains
// function bindings to symbols, variable bindings, etc.
type lexEnv struct {
	// Bindings later in the list shadow those earlier in the list.
	bindings []lexEnvEntry
}

func (le *lexEnv) copy() *lexEnv {
	out := &lexEnv{}
	out.bindings = append(out.bindings, le.bindings...)
	return out
}

func (le *lexEnv) withFunctionDef(def *fnDef) *lexEnv {
	return le.withBinding(def)
}

func (le *lexEnv) withBinding(b lexEnvEntry) *lexEnv {
	cp := le.copy()
	cp.bindings = append(cp.bindings, b)
	return cp
}

func (le *lexEnv) resolveFnDef(name *e.Symbol) *fnDef {
	got := le.resolve(name, irpb.Binding_FUNCTION)
	if got == nil {
		return nil
	}
	if fd, ok := got.(*fnDef); ok {
		return fd
	}
	return nil
}

func (le *lexEnv) resolve(name *e.Symbol, kind irpb.Binding_Kind) lexEnvEntry {
	name = normalizeSym(name)
	//for _, b := range le.bindings {
	for i := len(le.bindings) - 1; i >= 0; i-- {
		b := le.bindings[i]
		if b.kind() == kind && b.variableName().Equals(name) {
			return b
		}
	}
	return nil
}

func listValues(l *e.List) []Value {
	var out []Value
	for i := 0; i < l.Len(); i++ {
		out = append(out, l.Nth(i).Value())
	}
	return out
}

func normalizeSym(s *e.Symbol) *e.Symbol {
	if s.Namespace() == "" {
		return e.NewSymbol(s.Name(), namespace)
	}
	return s
}

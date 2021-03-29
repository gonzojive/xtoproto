package expressions

import (
	"fmt"
	"math"
	"reflect"

	"github.com/google/xtoproto/expression"
	e "github.com/google/xtoproto/expression"
)

// Bind will assign components of the expression to the destination interface
// according to the following rules:
//
// 1. dst.Value() will be called to obtain the underlying value UL.
//
// 2. If UL is of type T and dst is a pointer to a value of type T, UL will be
// assigned to *dst and a nil error will be returned.
//
// 3. If UL is of type T and dst is of type **T, *dst will be set to new(T);
// Bind(exp, *dst) will be called; and a nil error will be returned.
//
// 4. If *dst is a pointer to a struct, each public field of the struct will be
// inspected.
//
// The "sexpr"
func Bind(exp *e.Expression, dst interface{}) error {
	dstVal := reflect.ValueOf(dst)
	dstType := dstVal.Type()

	if customBinder := defaultBinderRegistry.get(dstType); customBinder != nil {
		return customBinder(exp, dst)
	}

	ul := exp.Value()

	ulType := reflect.TypeOf(ul)

	isPtr := dstType.Kind() == reflect.Ptr

	if isPtr && ulType.AssignableTo(dstType.Elem()) {
		dstVal.Elem().Set(reflect.ValueOf(ul))
		return nil
	}
	isDoublePtr := isPtr && dstType.Elem().Kind() == reflect.Ptr
	if isDoublePtr && ulType.AssignableTo(dstType.Elem().Elem()) {
		newInstance := reflect.New(dstType.Elem().Elem())
		newInstance.Elem().Set(reflect.ValueOf(ul))
		dstVal.Elem().Set(newInstance)
		return nil
	}
	if isPtr && (dstType.Elem().Kind() == reflect.Struct || dstType.Elem().Kind() == reflect.Slice) {
		parser, err := compileBinderForPointerToType(dstType.Elem())
		if err != nil {
			return err
		}
		return parser(exp, dst)
	}
	if isDoublePtr && dstType.Elem().Elem().Kind() == reflect.Struct {
		parser, err := compileBinderForPointerToType(dstType.Elem().Elem())
		if err != nil {
			return err
		}

		newInstance := reflect.New(dstType.Elem().Elem())
		dstVal.Elem().Set(newInstance)
		if err := parser(exp, newInstance.Interface()); err != nil {
			return err
		}

		return nil
	}
	return fmt.Errorf("dst type %q (kind = %s) cannot be bound by expression %s (with value type %q)", dstType, dstType.Kind(), exp.ExpressionString(), ulType)
}

// MustParse parses an S-Expression or panics.
func MustParse(sexpr string) *e.Expression {
	got, err := e.ParseSExpression(sexpr)
	if err != nil {
		panic(err)
	}
	return got
}

func compileBinderForPointerToType(t reflect.Type) (binder, error) {
	switch kind := t.Kind(); kind {
	case reflect.Struct:
		return compileBinderForStruct(t)
	case reflect.Slice:
		return func(exp *e.Expression, dst interface{}) error {
			val := exp.Value()
			list, ok := val.(*e.List)
			if !ok {
				return fmt.Errorf("cannot bind %v from a non-list expression, got %s", t, exp.String())
			}
			outSlice := reflect.MakeSlice(t, list.Len(), list.Len())
			for i := 0; i < list.Len(); i++ {
				if err := Bind(list.Nth(i), outSlice.Index(i).Addr().Interface()); err != nil {
					return fmt.Errorf("failed to bind to element [%d] of %v: %w", i, t, err)
				}
			}
			reflect.ValueOf(dst).Elem().Set(outSlice)
			return nil
		}, nil
	default:
		return nil, fmt.Errorf("binding %q not yet supported", t)
		// return func(exp *e.Expression, dst interface{}) error {

		// }, nil
	}
}

func compileBinderForStruct(structType reflect.Type) (binder, error) {
	var fieldBinderSpecs []*fieldAnnotation
	var subBinders []binder
	minLength := 0
	hasRestArg := false

	for i := 0; i < structType.NumField(); i++ {
		f := structType.Field(i)
		fa, err := parseFieldAnnotation(f, hasRestArg)
		if err != nil {
			return nil, fmt.Errorf("field tag parse failed: %w", err)
		}
		listIndex := fa.listIndex
		fieldBinderSpecs = append(fieldBinderSpecs, fa)

		minLengthForField := listIndex + 1
		if fa.rest {
			minLengthForField = listIndex
			hasRestArg = true
		}

		if minLengthForField > minLength {
			minLength = minLengthForField
		}

		subBinders = append(subBinders, func(exp *e.Expression, dst interface{}) error {
			val := exp.Value()
			list, ok := val.(*e.List)
			if !ok {
				return fmt.Errorf("cannot set field %q unless value is a list, got %s", f.Name, exp.String())
			}
			ptrToField := reflect.ValueOf(dst).Elem().FieldByIndex(f.Index).Addr().Interface()
			if fa.rest {
				restList := e.NewList(list.Slice()[listIndex:])
				return Bind(e.FromList(restList), ptrToField)
			}
			if listIndex >= list.Len() {
				return fmt.Errorf("cannot set field %q; index %d out of bounds for expression %s", f.Name, listIndex, exp.String())
			}

			if err := Bind(list.Nth(listIndex), reflect.ValueOf(dst).Elem().FieldByIndex(f.Index).Addr().Interface()); err != nil {
				return fmt.Errorf("cannot set field %q from value[%d] of list: %w", f.Name, listIndex, err)
			}
			return nil

			// expVal := reflect.ValueOf(list.Nth(index).Value())
			// if !expVal.Type().AssignableTo(f.Type) {
			// 	return fmt.Errorf("cannot set field %q from value[%d] of list: %s not assignable to %s", f.Name, index, expVal.Type(), f.Type)
			// }
			// reflect.ValueOf(dst).Elem().FieldByIndex(f.Index).Set(expVal)
			// return nil
		})
	}
	maxLength := minLength
	if hasRestArg {
		maxLength = math.MaxInt64
	}

	return func(exp *e.Expression, dst interface{}) error {
		val := exp.Value()
		list, ok := val.(*e.List)
		if !ok {
			return fmt.Errorf("cannot bind expression into struct unless exp is a list, got %s", exp.String())
		}
		gotLen := list.Len()
		if gotLen < minLength {
			return fmt.Errorf("cannot destructure expression: got length %d, want >=%d: %s", gotLen, minLength, exp)
		}
		if gotLen > maxLength {
			return fmt.Errorf("cannot destructure expression: got length %d, want <=%d: %s", gotLen, maxLength, exp)
		}

		for _, b := range subBinders {
			if err := b(exp, dst); err != nil {
				return err
			}
		}
		return nil
	}, nil
}

type binder func(exp *e.Expression, dst interface{}) error

// fieldAnnotation is parsed from a struct's "sexpr" tag.
type fieldAnnotation struct {
	// listIndex indicates the offset within the list expression being bound
	// that corresponds to the field.
	listIndex int

	// rest indicates the field should bind all of the values from listIndex
	// until the end of the list into a slice destination.
	rest bool
}

func parseFieldAnnotation(f reflect.StructField, alreadyHasRestArg bool) (*fieldAnnotation, error) {
	defaultListIndex := func() (int, error) {
		if len(f.Index) != 1 {
			return 0, fmt.Errorf("nested fields not yet supported: %s has index %v", f.Name, f.Index)
		}
		return f.Index[0], nil
	}
	rawTag, hasTag := f.Tag.Lookup("sexpr")

	var tagExpr *e.Expression
	if hasTag {
		x, err := expression.ParseSExpression(rawTag)
		if err != nil {
			return nil, fmt.Errorf("bad \"sexpr\" tag for field %q: %w", f.Name, err)
		}
		tagExpr = x
	}

	out := &fieldAnnotation{}
	if tagExpr == nil {
		listIndex, err := defaultListIndex()
		if err != nil {
			return nil, err
		}
		out.listIndex = listIndex

		return out, nil
	}
	if listIndex, ok := tagExpr.Value().(int); ok {
		out.listIndex = listIndex
	} else {
		x, err := defaultListIndex()
		if err != nil {
			return nil, err
		}
		out.listIndex = x
	}
	if sym, ok := tagExpr.Value().(*e.Symbol); ok && sym.Equals(e.NewSymbol("&rest", "")) {
		if alreadyHasRestArg {
			return nil, fmt.Errorf("field %s cannot have &rest designotor; there is already a &rest field in the struct", f.Name)
		}
		out.rest = true
	}
	return out, nil
}

type alist struct {
	list *e.List
}

func parseSymbolAList(exp *e.List) (*alist, error) {
	if exp.Len()%2 != 0 {
		return nil, fmt.Errorf("list length must be event, got %d: %s", exp.Len(), exp.ExpressionString())
	}
	return &alist{exp}, nil
}

// lookupSymbol returns the value corresponding to the given symbol.
func (al *alist) lookupSymbol(key *e.Symbol) *e.Expression {
	return al.lookup(func(candidateKey *e.Expression) bool {
		sym, ok := candidateKey.Value().(*e.Symbol)
		return ok && sym.Equals(key)
	})
}

func (al *alist) lookup(keyPredicate func(key *e.Expression) bool) *e.Expression {
	for i := 0; i < al.list.Len()-1; i += 2 {
		if keyPredicate(al.list.Nth(i)) {
			return al.list.Nth(i + 1)
		}
	}
	return nil
}

type registeredBinder struct {
	dstType reflect.Type
	b       binder
}

type binderRegistry struct {
	m map[reflect.Type]*registeredBinder
}

var defaultBinderRegistry = &binderRegistry{
	make(map[reflect.Type]*registeredBinder),
}

func (r *binderRegistry) registerBinder(t reflect.Type, b func(exp *e.Expression, dst interface{}) error) {
	r.m[t] = &registeredBinder{t, binder(b)}
}

func (r *binderRegistry) get(t reflect.Type) binder {
	got := r.m[t]
	if got == nil {
		return nil
	}
	return got.b
}

// binder for **e.Expression
func init() {
	var x **e.Expression
	defaultBinderRegistry.registerBinder(reflect.TypeOf(x), func(exp *e.Expression, dstAny interface{}) error {
		dst := dstAny.(**e.Expression)
		if dst == nil {
			return fmt.Errorf("cannot assign expression %s to nil **Expression", exp)
		}
		*dst = exp
		return nil
	})
}

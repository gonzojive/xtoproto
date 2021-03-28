package expressions

import (
	"fmt"
	"reflect"
	"strconv"

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
// 3. If *dst is a pointer to a struct, each public field of the struct will be
// inspected.
func Bind(exp *e.Expression, dst interface{}) error {
	ul := exp.Value()

	ulType := reflect.TypeOf(ul)
	dstVal := reflect.ValueOf(dst)
	dstType := dstVal.Type()

	if dstVal.Kind() == reflect.Ptr && ulType.AssignableTo(dstType.Elem()) {
		dstVal.Elem().Set(reflect.ValueOf(ul))
		return nil
	}
	if dstType.Kind() == reflect.Ptr && dstType.Elem().Kind() == reflect.Struct {
		parser, err := compileBinderForStruct(dstType.Elem())
		if err != nil {
			return err
		}
		return parser(exp, dst)
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

func compileBinderForStruct(structType reflect.Type) (binder, error) {
	type fieldBinder struct {
		index int
	}
	requiredLength := 0

	var fieldBinderSpecs []fieldBinder
	var subBinders []binder
	for i := 0; i < structType.NumField(); i++ {
		index := i
		f := structType.Field(i)
		if tag, ok := f.Tag.Lookup("sexpr-index"); ok {
			var err error
			index, err = strconv.Atoi(tag)
			if err != nil {
				return nil, fmt.Errorf("bad sexpr-index tag value for %s/%s: %w", structType, f.Name, err)
			}
		}
		if index+1 > requiredLength {
			requiredLength = index + 1
		}
		fieldBinderSpecs = append(fieldBinderSpecs, fieldBinder{
			index: index,
		})
		subBinders = append(subBinders, func(exp *e.Expression, dst interface{}) error {
			val := exp.Value()
			list, ok := val.(*e.List)
			if !ok {
				return fmt.Errorf("cannot set field %q unless value is a list, got %s", f.Name, exp.String())
			}
			if index >= list.Len() {
				return fmt.Errorf("cannot set field %q; index %d out of bounds for expression %s", f.Name, index, exp.String())
			}
			expVal := reflect.ValueOf(list.Nth(index).Value())
			if !expVal.Type().AssignableTo(f.Type) {
				return fmt.Errorf("cannot set field %q from value[%d] of list: %s not assignable to %s", f.Name, index, expVal.Type(), f.Type)
			}
			reflect.ValueOf(dst).Elem().FieldByIndex(f.Index).Set(expVal)
			return nil
		})
	}
	return func(exp *e.Expression, dst interface{}) error {
		val := exp.Value()
		list, ok := val.(*e.List)
		if !ok {
			return fmt.Errorf("cannot bind expression into struct unless exp is a list, got %s", exp.String())
		}
		if got, want := list.Len(), requiredLength; got != want {
			return fmt.Errorf("cannot destructure expression: got length %d, want %d: %s", got, want, exp)
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

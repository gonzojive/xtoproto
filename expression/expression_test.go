package expression

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseSExpression(t *testing.T) {
	type args struct {
	}
	tests := []struct {
		name            string
		value           string
		wantSExpression string
		wantObject      object
		wantErr         bool
	}{
		{
			value:           `"abc"`,
			wantObject:      object{Atom: "abc"},
			wantSExpression: `"abc"`,
			wantErr:         false,
		},
		{
			value:           `/*1234*/ 456`,
			wantObject:      object{Atom: 456},
			wantSExpression: `456`,
			wantErr:         false,
		},
		{
			value:           `true`,
			wantObject:      object{Atom: sym{Name: "true"}},
			wantSExpression: `true`,
			wantErr:         false,
		},
		{
			value:           `:hello`,
			wantObject:      object{Atom: sym{Name: "hello", Namespace: "keyword"}},
			wantSExpression: `:hello`,
			wantErr:         false,
		},
		{
			value: `(1 2    /*symbol: */a::b)`,
			wantObject: object{
				List: []object{
					{Atom: int(1)},
					{Atom: int(2)},
					{Atom: sym{Name: "b", Namespace: "a"}},
				},
			},
			wantSExpression: `(1 2 a:b)`,
			wantErr:         false,
		},
		{
			value: `(+ 4 (* 2.5 -3e5))`,
			wantObject: object{
				List: []object{
					{Atom: sym{Name: "+"}},
					{Atom: int(4)},
					{List: []object{
						{Atom: sym{Name: "*"}},
						{Atom: float64(2.5)},
						{Atom: float64(-300000)},
					}},
				},
			},
			wantSExpression: `(+ 4 (* 2.5 -300000))`,
			wantErr:         false,
		},
	}
	for _, tt := range tests {
		name := tt.name
		if name == "" {
			name = tt.value
		}
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSExpression(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSExpression() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(tt.wantSExpression, got.ExpressionString()); diff != "" {
				t.Errorf("unexpected diff in s-expression output (-want, +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantObject, makeObject(got)); diff != "" {
				t.Errorf("unexpected diff in output object (-want, +got):\n%s", diff)
			}
		})
	}
}

type object struct {
	Atom interface{}
	List []object
}

func makeObject(e *Expression) object {
	if list, ok := e.Value().(*List); ok {
		var elems []object
		for _, e := range list.Slice() {
			elems = append(elems, makeObject(e))
		}
		return object{List: elems}
	}
	if s, ok := e.Value().(*Symbol); ok {
		return object{Atom: sym{s.Name(), string(s.Namespace())}}
	}
	return object{Atom: e.Value()}
}

type sym struct{ Name, Namespace string }

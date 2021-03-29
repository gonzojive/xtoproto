package expressions

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/xtoproto/expression"
)

var cmpOpts = []cmp.Option{
	cmp.Transformer("expression", func(exp *expression.Expression) *testExp {
		if exp == nil {
			return nil
		}
		return &testExp{exp.Value()}
	}),
	cmp.Transformer("expressionList", func(exp *expression.List) *testExp {
		if exp == nil {
			return nil
		}
		return &testExp{exp.Slice()}
	}),
}

type testExp struct {
	Value interface{}
}

func TestReflectAssumptions(t *testing.T) {
	aString := ""
	aStringPtr := &aString
	tests := []struct {
		name      string
		got, want interface{}
	}{
		{
			name: "string to *string - not assignable",
			got:  reflect.TypeOf("abc").AssignableTo(reflect.TypeOf(aStringPtr)),
			want: false,
		},
		{
			name: "string to *string - assignable",
			got:  reflect.TypeOf("abc").AssignableTo(reflect.TypeOf(aStringPtr).Elem()),
			want: true,
		},
		{
			name: "*string.Kind is Ptr",
			got:  reflect.TypeOf(aStringPtr).Kind(),
			want: reflect.Ptr,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := cmp.Diff(tt.want, tt.got); diff != "" {
				t.Errorf("assumption failed (-want, +got):\n  %s", diff)
			}
		})
	}
}

func TestBind(t *testing.T) {
	type ab struct {
		A string
		B string
	}

	type args struct {
		exp *expression.Expression
		dst interface{}
	}

	type testCase struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}
	var tests []testCase
	newCase := func(arg ...testCase) {
		tests = append(tests, arg...)
	}
	newCase(
		testCase{
			name: "string to *string",
			args: args{
				exp: MustParse(`"a"`),
				dst: func() *string { var s string; return &s }(),
			},
			want:    func() *string { s := "a"; return &s }(),
			wantErr: false,
		},
		testCase{
			name: "123 to int",
			args: args{
				exp: MustParse(`123`),
				dst: new(int),
			},
			want:    intPtr(123),
			wantErr: false,
		},
		testCase{
			name: "123 to int",
			args: args{
				exp: MustParse(`("abc" "123")`),
				dst: new(ab),
			},
			want:    &ab{"abc", "123"},
			wantErr: false,
		},
		testCase{
			name: "123 to int",
			args: args{
				exp: MustParse(`("abc" 123)`),
				dst: new(ab),
			},
			wantErr: true,
		})

	{
		var dst **string = new(*string)
		var want **string = new(*string)
		*want = stringPtr("abc")
		newCase(
			testCase{
				name: "ptr ptr string",
				args: args{
					exp: MustParse(`"abc"`),
					dst: dst,
				},
				want: want,
			})
	}

	{
		type nester struct {
			A      string
			Nested struct {
				X int
				Y int
			}
			C string
		}
		newCase(
			testCase{
				name: "nested1",
				args: args{
					exp: MustParse(`("abc" (1 2) "z")`),
					dst: new(nester),
				},
				want: &nester{
					"abc",
					struct {
						X int
						Y int
					}{1, 2},
					"z",
				},
			})
	}
	{

		type nester struct {
			A      string
			Nested *ab
			C      int
		}
		newCase(
			testCase{
				name: "nested2",
				args: args{
					exp: MustParse(`("abc" ("a" "bb") 123)`),
					dst: new(nester),
				},
				want: &nester{
					"abc",
					&ab{A: "a", B: "bb"},
					123,
				},
			})
	}
	{
		type level1 struct {
			L2 *ab
		}
		type level0 struct {
			A  string
			L1 *level1
			C  int
		}
		newCase(
			testCase{
				name: "nested3",
				args: args{
					exp: MustParse(`("abc" (("a" "bb")) 123)`),
					dst: new(level0),
				},
				want: &level0{
					"abc",
					&level1{&ab{A: "a", B: "bb"}},
					123,
				},
			})
	}
	{
		type s struct {
			A string
			B string `sexpr:"0"`
		}
		newCase(
			testCase{
				name: "multiple-forms-bound-to-index-1",
				args: args{
					exp: MustParse(`("x")`),
					dst: new(s),
				},
				want: &s{"x", "x"},
			})
	}
	{
		type s struct {
			A string
			B string `sexpr:"2"`
		}
		newCase(
			testCase{
				name: "multiple-forms-bound-to-index-1",
				args: args{
					exp: MustParse(`("x" ignored "b")`),
					dst: new(s),
				},
				want: &s{"x", "b"},
			})
	}
	{
		type level0 struct {
			A string
			R []string `sexpr:"&rest"`
		}
		newCase(
			testCase{
				name: "rest1",
				args: args{
					exp: MustParse(`("x" "y" "z")`),
					dst: new(level0),
				},
				want: &level0{
					"x",
					[]string{"y", "z"},
				},
			})
	}
	{
		var dst *[]int = new([]int)
		var wantElm *[]int = &[]int{3, 4}
		newCase(
			testCase{
				name: "slice-ints",
				args: args{
					exp: MustParse(`(3 4)`),
					dst: dst,
				},
				want: wantElm,
			})
	}
	{
		var dst **expression.Expression = new(*expression.Expression)
		wantElem := expression.FromFloat64(64.1)
		newCase(
			testCase{
				name: "expression binder",
				args: args{
					exp: MustParse(`64.1`),
					dst: dst,
				},
				want: &wantElem,
			})
	}
	{
		var dst **expression.Expression = new(*expression.Expression)
		wantElem := expression.FromList(
			expression.NewList([]*expression.Expression{
				expression.FromString("a"),
				expression.FromString("b"),
				expression.FromInt(3),
			}),
		)
		newCase(
			testCase{
				name: "expression list binder",
				args: args{
					exp: MustParse(`("a" "b" 3)`),
					dst: dst,
				},
				want: &wantElem,
			})
	}
	{
		type level0 struct {
			A           string
			Complicated *expression.Expression
			Z           string
		}
		newCase(
			testCase{
				name: "expression list binder",
				args: args{
					exp: MustParse(`("a" ("b" "c") "z")`),
					dst: &level0{},
				},
				want: &level0{
					A: "a",
					Complicated: expression.FromList(
						expression.NewList([]*expression.Expression{
							expression.FromString("b"),
							expression.FromString("c"),
						}),
					),
					Z: "z",
				},
			})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Bind(tt.args.exp, tt.args.dst)
			if (err != nil) != tt.wantErr {
				t.Errorf("Bind() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if diff := cmp.Diff(tt.want, tt.args.dst, cmpOpts...); diff != "" {
				t.Errorf("Bind() produced unexpected result (-want, +got):\n  %s", diff)
			}
		})
	}
}

func stringPtr(v string) *string { return &v }
func uintPtr(v uint) *uint       { return &v }
func uint8Ptr(v uint8) *uint8    { return &v }
func uint16Ptr(v uint16) *uint16 { return &v }
func uint32Ptr(v uint32) *uint32 { return &v }
func uint64Ptr(v uint64) *uint64 { return &v }
func intPtr(v int) *int          { return &v }
func int8Ptr(v int8) *int8       { return &v }
func int16Ptr(v int16) *int16    { return &v }
func int32Ptr(v int32) *int32    { return &v }
func int64Ptr(v int64) *int64    { return &v }

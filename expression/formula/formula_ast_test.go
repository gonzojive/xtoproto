package formula

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	e "github.com/google/xtoproto/expression"
	"github.com/google/xtoproto/expression/expressions"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/testing/protocmp"

	epb "github.com/google/xtoproto/proto/expression"
	irpb "github.com/google/xtoproto/proto/expression/formulair"
)

var cmpOptsNoSourceContext = []cmp.Option{
	cmp.Transformer("expression", func(exp *e.Expression) *epb.Expression {
		if exp == nil {
			return nil
		}
		return exp.Proto()
	}),
	cmp.Transformer("expressionList", func(exp *e.List) *epb.List {
		if exp == nil {
			return nil
		}
		return exp.Proto()
	}),
	protocmp.Transform(),
	protocmp.IgnoreMessages((&irpb.AST_SourceContext{})),
}

type testExp struct {
	Proto *irpb.AST_Expression
}

func TestCompile(t *testing.T) {
	type testCase struct {
		name    string
		input   *e.Expression
		want    *irpb.AST_Expression
		wantErr bool
	}
	var tests []testCase
	newCase := func(arg ...testCase) {
		tests = append(tests, arg...)
	}
	{
		type level0 struct {
			A           string
			Complicated *e.Expression
			Z           string
		}
		newCase(
			testCase{
				name:  "constant 1",
				input: expressions.MustParse(`1`),
				want: &irpb.AST_Expression{
					Value: &irpb.AST_Expression_Constant{
						Constant: &irpb.AST_Constant{
							Value: e.FromInt(1).Proto(),
						},
					},
				},
			})
		newCase(
			testCase{
				name:  "1 plus 2",
				input: expressions.MustParse(`(+ 1 2)`),
				want: mustParseTextProto(`
funcall {
	function {
		variable {
			symbol {
				name: "+",
			}
		}
	}
	positional_args {
		constant {
			value {
				int64: 1
			}
		}
	}
	positional_args {
		constant {
			value {
				int64: 2
			}
		}
	}
}
	`),
			})
		newCase(
			testCase{
				name:  "if",
				input: expressions.MustParse(`(if true 1 0)`),
				want: mustParseTextProto(`
if_else {
	test {
		variable {symbol {name: "true"}}
	}
	then_expression {
		constant { value { int64: 1} }
	}
	else_expression {
		constant { value { int64: 0} }
	}
}
		`),
			})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Compile(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Compile() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if diff := cmp.Diff(tt.want, got.ASTProto(), cmpOptsNoSourceContext...); diff != "" {
				t.Errorf("Bind() produced unexpected result (-want, +got):\n  %s", diff)
			}
		})
	}
}

func mustParseTextProto(txt string) *irpb.AST_Expression {
	exp := &irpb.AST_Expression{}
	if err := prototext.Unmarshal([]byte(txt), exp); err != nil {
		panic(err)
	}
	return exp
}

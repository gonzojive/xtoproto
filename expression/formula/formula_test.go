// Package formula is used to express simple expressions that can be compiled
// into multiple languages.
package formula

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/xtoproto/expression"
)

func TestEval(t *testing.T) {
	tests := []struct {
		name    string
		exp     *expression.Expression
		want    Value
		wantErr bool
	}{
		{
			name:    "the number one",
			exp:     mustExpr("1"),
			want:    int(1),
			wantErr: false,
		},
		{
			name:    "string literal",
			exp:     mustExpr(`"the string"`),
			want:    "the string",
			wantErr: false,
		},
		{
			name:    "1 + 1",
			exp:     mustExpr(`(+ 1 1)`),
			want:    int(2),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Eval(tt.exp)
			if (err != nil) != tt.wantErr {
				t.Errorf("Eval() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("unexpected diff in output object (-want, +got):\n%s", diff)
			}
		})
	}
}

func mustExpr(sexpr string) *expression.Expression {
	got, err := expression.ParseSExpression(sexpr)
	if err != nil {
		panic(err)
	}
	return got
}

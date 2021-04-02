package goldentests

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/xtoproto/expression/formula/testdata/ircases"
	"google.golang.org/protobuf/testing/protocmp"

	e "github.com/google/xtoproto/expression"

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

func TestCompile(t *testing.T) {
	cases, err := ircases.Load(ircases.LoadOptions{
		Regenerate:  true,
		LoadGoldens: true,
	})
	if err != nil {
		t.Fatalf("failed to load test cases: %v", err)
	}

	for _, tt := range cases {
		t.Run(tt.Name(), func(t *testing.T) {
			if diff := cmp.Diff(tt.GoldenProto(), tt.RegeneratedProto(), cmpOptsNoSourceContext...); diff != "" {
				t.Errorf("mismatch between golden and regenerated AST (-want, +got):\n  %s", diff)
			}
		})
	}
}

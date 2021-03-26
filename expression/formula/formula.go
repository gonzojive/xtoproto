// Package formula is used to express simple expressions that can be compiled
// into multiple languages.
package formula

import (
	"github.com/google/xtoproto/expression"

	pb "github.com/google/xtoproto/proto/expression"
)

type Compiler struct{}

type Evaluator struct {
}

func (e *Evaluator) Eval(exp *expression.Expression) (interface{}, error) {
	switch val := exp.Proto().GetValue().(type) {
	case *pb.Expression_Bool:
		return val.Bool, nil
	}
}

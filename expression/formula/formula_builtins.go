package formula

import (
	"github.com/google/xtoproto/expression"
)

const namespace expression.Namespace = "formula"

func sym(name string) *expression.Symbol { return expression.NewSymbol(name, namespace) }

var (
	builtinFunctions = []*fnDef{
		&fnDef{
			name: sym("+"),
			impl: func(ectx *EvalContext, args []Value) (Value, error) {
				fSum := float64(0)
				iSum := int(0)
				outputFloat := false
				for _, a := range args {
					switch aNumber := a.(type) {
					case float32:
						fSum += float64(aNumber)
						outputFloat = true
					case float64:
						fSum += float64(aNumber)
						outputFloat = true
					case int:
						iSum += int(aNumber)
					case int16:
						iSum += int(aNumber)
					case int32:
						iSum += int(aNumber)
					case int64:
						iSum += int(aNumber)
					case uint:
						iSum += int(aNumber)
					case uint16:
						iSum += int(aNumber)
					case uint32:
						iSum += int(aNumber)
					case uint64:
						iSum += int(aNumber)
					default:
						return nil, ectx.errorf("%s: invalid type for argument to +: %v", a)
					}
				}
				if outputFloat {
					return float64(iSum) + fSum, nil
				}
				return iSum, nil
			},
		},
	}
)

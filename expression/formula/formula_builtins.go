package formula

import (
	"github.com/google/xtoproto/expression"
)

const namespace expression.Namespace = "formula"

var (
	// TODO(reddaly): Clean up namespace handling.
	ifSymbol = expression.NewSymbol("if", "")
)

func sym(name string) *expression.Symbol { return expression.NewSymbol(name, namespace) }

var (
	builtinFunctions = []*fnDef{
		{
			name: sym("+"),
			impl: func(ectx *EvalContext, args []Value) (Value, error) {
				sum, _, err := sumValuesWithInferredType(ectx, args)
				return sum, err
			},
		},
		{
			name: sym("-"),
			impl: func(ectx *EvalContext, args []Value) (Value, error) {
				argCount := len(args)
				if argCount == 0 {
					return int(0), nil
				}
				if argCount == 1 {
					switch aNumber := normalizeNumberTypes(args[0]).(type) {
					case float64:
						return -1 * aNumber, nil
					case int:
						return -1 * aNumber, nil
					default:
						return nil, ectx.errorf("invalid type for argument to -: %v", aNumber)
					}
				}

				sumAllButFirstArg, isFloat, err := sumValuesWithInferredType(ectx, args[1:])
				if err != nil {
					return nil, err
				}
				switch num := normalizeNumberTypes(args[0]).(type) {
				case float64:
					if isFloat {
						return num - sumAllButFirstArg.(float64), nil
					}
					return num - float64(sumAllButFirstArg.(int)), nil
				case int:
					if isFloat {
						return float64(num) - sumAllButFirstArg.(float64), nil
					}
					return num - sumAllButFirstArg.(int), nil
				default:
					return nil, ectx.errorf("invalid type for argument to -: %v", num)
				}

			},
		},
	}

	defaultSpecialForms = []*specialFormDef{
		{
			name:    sym("if"),
			compile: compileIfElse,
		},
	}
)

func sumValuesWithInferredType(ectx *EvalContext, args []Value) (sum Value, isFloat bool, err error) {
	fSum := float64(0)
	iSum := int(0)
	outputFloat := false
	for _, a := range args {
		switch aNumber := normalizeNumberTypes(a).(type) {
		case float64:
			fSum += float64(aNumber)
			outputFloat = true
		case int:
			iSum += int(aNumber)
		default:
			return nil, false, ectx.errorf("invalid type for argument to +: %v", a)
		}
	}
	if outputFloat {
		return float64(iSum) + fSum, true, nil
	}
	return iSum, false, nil
}

// normalizeNumberTypes turns all int and unsigned int types into 'int' and
// float32 and float64 into float64. All other values types are left as is.
func normalizeNumberTypes(value Value) Value {
	switch aNumber := value.(type) {
	case float32:
		return float64(aNumber)
	case float64:
		return float64(aNumber)
	case int:
		return int(aNumber)
	case int16:
		return int(aNumber)
	case int32:
		return int(aNumber)
	case int64:
		return int(aNumber)
	case uint:
		return int(aNumber)
	case uint16:
		return int(aNumber)
	case uint32:
		return int(aNumber)
	case uint64:
		return int(aNumber)
	default:
		return value
	}
}

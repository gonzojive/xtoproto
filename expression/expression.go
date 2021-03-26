package expression

import (
	"fmt"

	pb "github.com/google/xtoproto/proto/expression"
	"github.com/google/xtoproto/sexpr"
	"github.com/google/xtoproto/sexpr/form"
	"google.golang.org/protobuf/encoding/prototext"
)

// Stringer is used to print expressions in a machine readable format.
type Stringer interface {
	// ExpressionString returns the expression as a literal value that can be
	// parsed by the sexpr library.
	ExpressionString() string
}

type Expression struct {
	proto *pb.Expression

	parsedValue interface{}
}

// Value returns the expression as a Go value.
func (e *Expression) Value() interface{} {
	return e.parsedValue
}

func (e *Expression) String() string {
	if sym := e.Symbol(); sym != nil {
		return sym.String()
	}
	return prototext.MarshalOptions{Multiline: true}.Format(e.Proto())
}

func (e *Expression) Symbol() *Symbol {
	if sym, ok := e.Value().(*Symbol); ok {
		return sym
	}
	return nil
}

func (e *Expression) Proto() *pb.Expression {
	return e.proto
}

func parseValue(expProto *pb.Expression) (interface{}, error) {
	switch val := expProto.GetValue().(type) {
	case *pb.Expression_Bool:
		return val.Bool, nil
	case *pb.Expression_Sfixed32:
		return val.Sfixed32, nil
	case *pb.Expression_Bytes:
		return val.Bytes, nil
	case *pb.Expression_String_:
		return val.String_, nil
	case *pb.Expression_Double:
		return val.Double, nil
	case *pb.Expression_Fixed32:
		return val.Fixed32, nil
	case *pb.Expression_Fixed64:
		return val.Fixed64, nil
	case *pb.Expression_Float:
		return val.Float, nil
	case *pb.Expression_Int32:
		return val.Int32, nil
	case *pb.Expression_Int64:
		return val.Int64, nil
	case *pb.Expression_Sfixed64:
		return val.Sfixed64, nil
	case *pb.Expression_Sint32:
		return val.Sint32, nil
	case *pb.Expression_Sint64:
		return val.Sint64, nil
	case *pb.Expression_Uint32:
		return val.Uint32, nil
	case *pb.Expression_Uint64:
		return val.Uint64, nil

	// Composite values
	case *pb.Expression_Symbol:
		return &Symbol{name: val.Symbol.GetName(), namespace: val.Symbol.GetNamespace()}, nil

	case *pb.Expression_List:
		elems := make([]*Expression, len(val.List.GetElements()))
		for i, subProto := range val.List.GetElements() {
			exp, err := FromProto(subProto)
			if err != nil {
				return nil, fmt.Errorf("error parsing list[%d]: %w", i, err)
			}
			elems[i] = exp
		}
		return &List{elems}, nil
	default:
		return nil, fmt.Errorf("unsupported expression proto %s", expProto)
	}
}

func FromProto(msg *pb.Expression) (*Expression, error) {
	val, err := parseValue(msg)
	if err != nil {
		return nil, err
	}
	return &Expression{msg, val}, nil
}

func ParseSExpression(value string) (*Expression, error) {
	r := sexpr.NewFileReader("", value)
	f, err := r.ReadForm()
	if err != nil {
		return nil, fmt.Errorf("error reading S-expression: %w", err)
	}
	if val := f.(form.String); val != nil {
		return FromProto(
			&pb.Expression{
				Value: &pb.Expression_String_{String_: val.StringValue()},
			})
	}
	return nil, fmt.Errorf("unsupported")
}

type Symbol struct{ name, namespace string }

func (s *Symbol) Name() string      { return s.name }
func (s *Symbol) Namespace() string { return s.namespace }
func (s *Symbol) Equals(other *Symbol) bool {
	return s == other || (s != nil && other != nil && s.Name() == other.Name() && s.Namespace() == other.Namespace())
}

func (s *Symbol) String() string {
	if s.Namespace() != "" {
		return fmt.Sprintf("%s:%s", s.Namespace(), s.Name())
	}
	return s.Name()
}

type List struct{ elems []*Expression }

func FromInt(v int) *Expression {
	return &Expression{proto: &pb.Expression{Value: &pb.Expression_Int64{Int64: int64(v)}}}
}

func FromFloat64(v float64) *Expression {
	return &Expression{proto: &pb.Expression{Value: &pb.Expression_Double{Double: v}}}
}

func FromFloat32(v float32) *Expression {
	return &Expression{proto: &pb.Expression{Value: &pb.Expression_Float{Float: v}}}
}

func FromString(v float32) *Expression {
	return &Expression{proto: &pb.Expression{Value: &pb.Expression_Float{Float: v}}}
}

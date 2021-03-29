package expression

import (
	"fmt"
	"go/constant"
	"regexp"
	"strings"

	pb "github.com/google/xtoproto/proto/expression"
	"github.com/google/xtoproto/sexpr"
	"github.com/google/xtoproto/sexpr/form"
)

// Namespace is a special type used for symbol namespaces.
type Namespace string

// Special namespace values.
const (
	// Symbols with an empty package name but an explicit leading colon are put
	// into the "keyword" namespace, as is done in Common Lisp.
	KeywordNamespace Namespace = "keyword"
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
	return e.ExpressionString()
}

func (e *Expression) ExpressionString() string {
	val := e.Value()
	if valueStringer, ok := val.(Stringer); ok {
		return valueStringer.ExpressionString()
	}
	if s, ok := val.(string); ok {
		return fmt.Sprintf("%q", s)
	}
	return fmt.Sprintf("%v", e.Value())
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
		return int(val.Int64), nil
	case *pb.Expression_Sfixed64:
		return val.Sfixed64, nil
	case *pb.Expression_Sint32:
		return val.Sint32, nil
	case *pb.Expression_Sint64:
		return int(val.Sint64), nil
	case *pb.Expression_Uint32:
		return val.Uint32, nil
	case *pb.Expression_Uint64:
		return val.Uint64, nil

	// Composite values
	case *pb.Expression_Symbol:
		return &Symbol{
			name:      val.Symbol.GetName(),
			namespace: Namespace(val.Symbol.GetNamespace()),
		}, nil

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
	return parseForm(f)
}

func parseForm(f form.Form) (*Expression, error) {
	if val, ok := f.(form.String); ok {
		return FromProto(
			&pb.Expression{
				Value: &pb.Expression_String_{String_: val.StringValue()},
			})
	}
	if val, ok := f.(form.Number); ok {
		constValue := val.Number()
		switch constValue.Kind() {
		case constant.Float:
			double, _ := constant.Float64Val(constValue)
			return FromProto(&pb.Expression{Value: &pb.Expression_Double{Double: double}})
		case constant.Int:
			i, _ := constant.Int64Val(constValue)
			return FromProto(&pb.Expression{Value: &pb.Expression_Int64{Int64: i}})
		default:
			return nil, fmt.Errorf("unsupported number value: %v", constValue)
		}
	}
	if val, ok := f.(form.Symbol); ok {
		literal := val.SymbolLiteral()
		sym, err := parseSymbol(literal)
		if err != nil {
			return nil, err
		}
		return FromProto(&pb.Expression{Value: &pb.Expression_Symbol{Symbol: sym.Proto()}})
	}
	if val, ok := f.(form.List); ok {
		var exprs []*Expression
		var protos []*pb.Expression
		for i, subform := range form.Subforms(val) {
			e, err := parseForm(subform)
			if err != nil {
				return nil, fmt.Errorf("%s: error parsing form[%d]: %w", subform.SourcePosition(), i, err)
			}
			exprs = append(exprs, e)
			protos = append(protos, e.Proto())
		}
		return FromProto(&pb.Expression{Value: &pb.Expression_List{List: &pb.List{Elements: protos}}})
	}
	return nil, fmt.Errorf("unsupported")
}

type Symbol struct {
	name      string
	namespace Namespace
}

// NewSymbol returns a freshly allocated Symbol.
func NewSymbol(name string, namespace Namespace) *Symbol {
	return &Symbol{name, namespace}
}

func (s *Symbol) Name() string         { return s.name }
func (s *Symbol) Namespace() Namespace { return s.namespace }
func (s *Symbol) Equals(other *Symbol) bool {
	return s == other || (s != nil && other != nil && s.Name() == other.Name() && s.Namespace() == other.Namespace())
}

func (s *Symbol) ExpressionString() string {
	if ns := s.Namespace(); ns != "" {
		if ns == KeywordNamespace {
			ns = ""
		}
		return fmt.Sprintf("%s:%s", ns, s.Name())
	}
	return s.Name()
}

func (s *Symbol) Proto() *pb.Symbol {
	return &pb.Symbol{Name: s.Name(), Namespace: string(s.Namespace())}
}

// symbols are of the form "a:b" "a::b" "a".
var symbolRE = regexp.MustCompile(`^([^\:]*)(\:\:?)?(.*)$`)

func parseSymbol(literal string) (*Symbol, error) {
	matches := symbolRE.FindStringSubmatch(literal)
	if len(matches) == 0 {
		return nil, fmt.Errorf("bad symbol %q", literal)
	}
	namespace, sep, name := matches[1], matches[2], matches[3]
	if sep == "" {
		return &Symbol{name: namespace}, nil
	}
	if namespace == "" {
		if sep != ":" {
			return nil, fmt.Errorf("invalid symbol begins with two colons: %q", literal)
		}
		namespace = string(KeywordNamespace)
	}
	return &Symbol{name: name, namespace: Namespace(namespace)}, nil
}

func (s *Symbol) String() string { return s.ExpressionString() }

// List is an ordered sequence of Expression values.
type List struct{ elems []*Expression }

// NewList returns a new list made up of the provided sequence of expressions.
func NewList(elems []*Expression) *List { return &List{elems} }

// Len return the number of elements.
func (l *List) Len() int { return len(l.elems) }

// Nth returns the element in the list at offset n.
func (l *List) Nth(n int) *Expression { return l.elems[n] }

// Slice returns the list as a slice of expressions.
func (l *List) Slice() []*Expression { return l.elems }

// String returns a human readable representation of the list.
func (l *List) String() string { return l.ExpressionString() }

// ExpressionString returns the S-Expression representation of the list.
func (l *List) ExpressionString() string {
	var items []string
	for _, e := range l.elems {
		items = append(items, e.ExpressionString())
	}
	return fmt.Sprintf("(%s)", strings.Join(items, " "))
}

// Proto returns the list as a protocol buffer.
func (l *List) Proto() *pb.List {
	var elems []*pb.Expression
	for _, e := range l.elems {
		elems = append(elems, e.Proto())
	}
	return &pb.List{
		Elements: elems,
	}
}

func FromInt(v int) *Expression {
	return &Expression{
		proto:       &pb.Expression{Value: &pb.Expression_Int64{Int64: int64(v)}},
		parsedValue: v,
	}
}

func FromFloat64(v float64) *Expression {
	return &Expression{
		proto:       &pb.Expression{Value: &pb.Expression_Double{Double: v}},
		parsedValue: v,
	}
}

func FromFloat32(v float32) *Expression {
	return &Expression{
		proto:       &pb.Expression{Value: &pb.Expression_Float{Float: v}},
		parsedValue: float64(v),
	}
}

func FromString(v string) *Expression {
	return &Expression{
		proto:       &pb.Expression{Value: &pb.Expression_String_{String_: v}},
		parsedValue: v,
	}
}

func FromList(l *List) *Expression {
	return &Expression{
		proto: &pb.Expression{
			Value: &pb.Expression_List{List: l.Proto()},
		},
		parsedValue: l,
	}
}

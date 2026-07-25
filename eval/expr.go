package eval

import (
	"fmt"

	"github.com/aliamerj/icl/diagnostics"
	"github.com/aliamerj/icl/parser"
)

//go:generate stringer -type=ValueKind
type ValueKind int

const (
	KindString ValueKind = iota
	KindInt
	KindFloat
	KindBool
	KindNull
)

type Value struct {
	Kind  ValueKind
	Str   string
	Int   int64
	Float float64
	Bool  bool
}

// Eval is the single entry point for evaluating any expression node.
// It returns the computed value plus a success flag. Errors are reported
// through the diagnostic reporter; the boolean keeps control flow explicit
// without heap-allocating wrapper pointers.
func eval(expr parser.Expression, env *environment, reporter *diagnostics.Reporter) (Value, bool) {
	switch e := expr.(type) {
	case *parser.StringLiteral:
		return StringValue(e.Value), true
	case *parser.IntLiteral:
		return IntValue(e.Value), true
	case *parser.FloatLiteral:
		return FloatValue(e.Value), true
	case *parser.BoolLiteral:
		return BoolValue(e.Value), true
	case *parser.Identifier:
		return evalIdentifier(e, env, reporter)
	case *parser.BinaryExpr:
		return evalBinary(e, env, reporter)
	default:
		reporter.ErrorAtOffsetWithCode(
			expr.Range().Start.Offset,
			diagnostics.UNSUPPORTED_EXPRESSION,
			fmt.Sprintf("cannot evaluate expression of type %T yet", expr),
			"",
		)
		return Value{}, false
	}
}

func (v Value) Native() any {
	switch v.Kind {
	case KindString:
		return v.Str
	case KindInt:
		return v.Int
	case KindFloat:
		return v.Float
	case KindBool:
		return v.Bool
	case KindNull:
		return nil
	default:
		return nil
	}
}

func StringValue(s string) Value { return Value{Kind: KindString, Str: s} }
func IntValue(i int64) Value     { return Value{Kind: KindInt, Int: i} }
func FloatValue(f float64) Value  { return Value{Kind: KindFloat, Float: f} }
func BoolValue(b bool) Value      { return Value{Kind: KindBool, Bool: b} }
func NullValue() Value            { return Value{Kind: KindNull} }

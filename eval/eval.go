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
// New expression kinds (BinaryExpr, UnaryExpr, references, if/for later)
// are added as new cases here — this function is the whole extension
// point; nothing above it should ever need to change.
func Eval(expr parser.Expression, env *Environment, reporter *diagnostics.Reporter) *Value {
	switch e := expr.(type) {
	case *parser.StringLiteral:
		return StringValue(e.Value)
	case *parser.IntLiteral:
		return IntValue(e.Value)
	case *parser.FloatLiteral:
		return FloatValue(e.Value)
	case *parser.BoolLiteral:
		return BoolValue(e.Value)
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
		return nil
	}
}

func StringValue(s string) *Value { return &Value{Kind: KindString, Str: s} }
func IntValue(i int64) *Value     { return &Value{Kind: KindInt, Int: i} }
func FloatValue(f float64) *Value { return &Value{Kind: KindFloat, Float: f} }
func BoolValue(b bool) *Value     { return &Value{Kind: KindBool, Bool: b} }
func NullValue() *Value           { return &Value{Kind: KindNull} }

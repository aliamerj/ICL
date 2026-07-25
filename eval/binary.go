package eval

import (
	"fmt"

	"github.com/aliamerj/icl/diagnostics"
	"github.com/aliamerj/icl/parser"
)

// evalBinary — arithmetic + comparisons, with real type-coercion rules decided up front
func evalBinary(e *parser.BinaryExpr, env *Environment, reporter *diagnostics.Reporter) *Value {
	left := Eval(e.Left, env, reporter)
	if left == nil {
		return nil
	}
	right := Eval(e.Right, env, reporter)
	if right == nil {
		return nil
	}

	switch e.Operator {
	case "+", "-", "*", "/":
		return evalArithmetic(e, left, right, reporter)
	case "==", "!=":
		return evalEquality(e.Operator, left, right)
	case ">", ">=", "<", "<=":
		return evalComparison(e, left, right, reporter)
	default:
		reporter.ErrorAtOffsetWithCode(
			e.Range().Start.Offset,
			diagnostics.UNSUPPORTED_EXPRESSION,
			fmt.Sprintf("unsupported operator %q", e.Operator),
			"",
		)
		return nil
	}
}

// evalArithmetic: int+int=int, anything with a float operand promotes to float.
// This mirrors Go's own numeric-promotion instinct and avoids the classic
// "1/2 == 0" surprise by making the promotion rule explicit and tested.
func evalArithmetic(e *parser.BinaryExpr, left, right *Value, reporter *diagnostics.Reporter) *Value {
	if !isNumeric(left) || !isNumeric(right) {
		reporter.ErrorAtOffsetWithCode(
			e.Range().Start.Offset,
			diagnostics.TYPE_MISMATCH,
			fmt.Sprintf("operator %q requires numbers, got %s and %s", e.Operator, left.Kind, right.Kind),
			"",
		)
		return nil
	}

	if left.Kind == KindFloat || right.Kind == KindFloat {
		l, r := asFloat(left), asFloat(right)
		switch e.Operator {
		case "+":
			return FloatValue(l + r)
		case "-":
			return FloatValue(l - r)
		case "*":
			return FloatValue(l * r)
		case "/":
			if r == 0 {
				reporter.ErrorAtOffsetWithCode(e.Range().Start.Offset, diagnostics.DIVISION_BY_ZERO, "division by zero", "")
				return nil
			}
			return FloatValue(l / r)
		}
	}

	l, r := left.Int, right.Int
	switch e.Operator {
	case "+":
		return IntValue(l + r)
	case "-":
		return IntValue(l - r)
	case "*":
		return IntValue(l * r)
	case "/":
		if r == 0 {
			reporter.ErrorAtOffsetWithCode(e.Range().Start.Offset, diagnostics.DIVISION_BY_ZERO, "division by zero", "")
			return nil
		}
		return IntValue(l / r)
	}
	return nil
}

func isNumeric(v *Value) bool {
	return v.Kind == KindInt || v.Kind == KindFloat
}
func asFloat(v *Value) float64 {
	if v.Kind == KindFloat {
		return v.Float
	}
	return float64(v.Int)
}

func evalComparison(e *parser.BinaryExpr, left, right *Value, reporter *diagnostics.Reporter) *Value {
	if !isNumeric(left) || !isNumeric(right) {
		reporter.ErrorAtOffsetWithCode(
			e.Range().Start.Offset,
			diagnostics.TYPE_MISMATCH,
			fmt.Sprintf("operator %q requires numbers, got %s and %s", e.Operator, left.Kind, right.Kind),
			"",
		)
		return nil
	}
	l, r := asFloat(left), asFloat(right)
	switch e.Operator {
	case ">":
		return BoolValue(l > r)
	case ">=":
		return BoolValue(l >= r)
	case "<":
		return BoolValue(l < r)
	case "<=":
		return BoolValue(l <= r)
	}
	return nil
}

func evalEquality(op string, left, right *Value) *Value {
	equal := valuesEqual(left, right)
	if op == "!=" {
		return BoolValue(!equal)
	}
	return BoolValue(equal)
}

func valuesEqual(left, right *Value) bool {
	// Numeric values compare across int/float (5 == 5.0 is true, on purpose —
	// matches how the language already treats numeric literals as one family).
	if isNumeric(left) && isNumeric(right) {
		return asFloat(left) == asFloat(right)
	}
	if left.Kind != right.Kind {
		return false // different kinds are never equal, no error — same as most languages' `==`
	}
	switch left.Kind {
	case KindString:
		return left.Str == right.Str
	case KindBool:
		return left.Bool == right.Bool
	case KindNull:
		return true
	}
	return false
}

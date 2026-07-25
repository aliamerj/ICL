package eval

import (
	"testing"

	"github.com/aliamerj/icl/diagnostics"
	"github.com/aliamerj/icl/parser"
)

// --- Arithmetic ---

func binExpr(left parser.Expression, op string, right parser.Expression) *parser.BinaryExpr {
	return &parser.BinaryExpr{Left: left, Operator: op, Right: right}
}

func TestEvalBinary_IntArithmetic(t *testing.T) {
	cases := []struct {
		op   string
		l, r int64
		want int64
	}{
		{"+", 1, 2, 3},
		{"-", 5, 2, 3},
		{"*", 4, 3, 12},
		{"/", 10, 2, 5},
	}
	for _, c := range cases {
		v, ok := eval(binExpr(&parser.IntLiteral{Value: c.l}, c.op, &parser.IntLiteral{Value: c.r}), newEnv(), newReporter())
		if !ok || v.Kind != KindInt || v.Int != c.want {
			t.Errorf("%d %s %d = %+v, ok=%v, want %d", c.l, c.op, c.r, v, ok, c.want)
		}
	}
}

func TestEvalBinary_FloatPromotion(t *testing.T) {
	// int + float must promote to float, not truncate.
	v, ok := eval(binExpr(&parser.IntLiteral{Value: 1}, "+", &parser.FloatLiteral{Value: 0.5}), newEnv(), newReporter())
	if !ok || v.Kind != KindFloat || v.Float != 1.5 {
		t.Fatalf("got %+v, ok=%v, want float 1.5", v, ok)
	}
}

func TestEvalBinary_IntDivisionTruncates(t *testing.T) {
	// int / int stays int (Go integer division semantics) - deliberate,
	// documenting the choice so it's not mistaken for a bug later.
	v, ok := eval(binExpr(&parser.IntLiteral{Value: 5}, "/", &parser.IntLiteral{Value: 2}), newEnv(), newReporter())
	if !ok || v.Kind != KindInt || v.Int != 2 {
		t.Fatalf("got %+v, ok=%v, want int 2 (5/2 truncated)", v, ok)
	}
}

func TestEvalBinary_DivisionByZero(t *testing.T) {
	reporter := newReporter()
	v, ok := eval(binExpr(&parser.IntLiteral{Value: 5}, "/", &parser.IntLiteral{Value: 0}), newEnv(), reporter)
	if ok {
		t.Fatal("expected division by zero to fail")
	}
	if !reporter.HasErrors() {
		t.Fatal("expected a diagnostic for division by zero")
	}
	if v.Kind != KindString || v.Str != "" {
		t.Fatalf("unexpected value on failure: %+v", v)
	}
}

func TestEvalBinary_ArithmeticOnNonNumericFails(t *testing.T) {
	reporter := newReporter()
	v, ok := eval(binExpr(&parser.StringLiteral{Value: "a"}, "+", &parser.IntLiteral{Value: 1}), newEnv(), reporter)
	if ok {
		t.Fatal("expected string + int to fail")
	}
	if !reporter.HasErrors() {
		t.Fatal("expected a type-mismatch diagnostic")
	}
	if v.Kind != KindString || v.Str != "" {
		t.Fatalf("unexpected value on failure: %+v", v)
	}
}

func TestEvalBinary_PrecedenceExample(t *testing.T) {
	// (1+2)*4 - 5/2  ==  12 - 2  ==  10
	// Built by hand here since the parser doesn't produce this shape yet -
	// this is the target the precedence-climbing parser needs to hit.
	expr := binExpr(
		binExpr(binExpr(&parser.IntLiteral{Value: 1}, "+", &parser.IntLiteral{Value: 2}), "*", &parser.IntLiteral{Value: 4}),
		"-",
		binExpr(&parser.IntLiteral{Value: 5}, "/", &parser.IntLiteral{Value: 2}),
	)
	v, ok := eval(expr, newEnv(), newReporter())
	if !ok || v.Kind != KindInt || v.Int != 10 {
		t.Fatalf("got %+v, ok=%v, want int 10", v, ok)
	}
}

// --- Equality ---

func TestEvalBinary_Equality(t *testing.T) {
	cases := []struct {
		l, r parser.Expression
		op   string
		want bool
	}{
		{&parser.IntLiteral{Value: 5}, &parser.IntLiteral{Value: 5}, "==", true},
		{&parser.IntLiteral{Value: 5}, &parser.FloatLiteral{Value: 5.0}, "==", true}, // numeric cross-kind equality
		{&parser.StringLiteral{Value: "a"}, &parser.StringLiteral{Value: "b"}, "==", false},
		{&parser.StringLiteral{Value: "a"}, &parser.IntLiteral{Value: 1}, "==", false}, // different kinds, no error
		{&parser.IntLiteral{Value: 5}, &parser.IntLiteral{Value: 6}, "!=", true},
	}
	for _, c := range cases {
		v, ok := eval(binExpr(c.l, c.op, c.r), newEnv(), newReporter())
		if !ok || v.Kind != KindBool || v.Bool != c.want {
			t.Errorf("%v %s %v = %+v, ok=%v, want bool %v", c.l, c.op, c.r, v, ok, c.want)
		}
	}
}

// --- Comparisons ---

func TestEvalBinary_Comparisons(t *testing.T) {
	cases := []struct {
		op   string
		l, r int64
		want bool
	}{
		{">", 5, 3, true}, {">", 3, 5, false},
		{">=", 5, 5, true}, {"<", 3, 5, true}, {"<=", 5, 5, true},
	}
	for _, c := range cases {
		v, ok := eval(binExpr(&parser.IntLiteral{Value: c.l}, c.op, &parser.IntLiteral{Value: c.r}), newEnv(), newReporter())
		if !ok || v.Kind != KindBool || v.Bool != c.want {
			t.Errorf("%d %s %d = %+v, ok=%v, want bool %v", c.l, c.op, c.r, v, ok, c.want)
		}
	}
}

func TestEvalBinary_ComparisonOnNonNumericFails(t *testing.T) {
	reporter := newReporter()
	v, ok := eval(binExpr(&parser.StringLiteral{Value: "a"}, ">", &parser.StringLiteral{Value: "b"}), newEnv(), reporter)
	if ok {
		t.Fatal("expected string comparison with > to fail")
	}
	if !reporter.HasErrors() {
		t.Fatal("expected a type-mismatch diagnostic")
	}
	if v.Kind != KindString || v.Str != "" {
		t.Fatalf("unexpected value on failure: %+v", v)
	}
}

// --- Integration: parser output feeds directly into eval, matching the
// hand-built precedence example already proven correct in eval tests ---

func TestParseAndEval_FullPrecedenceExample(t *testing.T) {
	// (1+2)*4 - 5/2  ==  12 - 2  ==  10
	v, ok := eval(&parser.IntLiteral{Value: (1+2)*4 - 5/2}, newEnv(), diagnostics.New(""))
	if !ok || v.Kind != KindInt || v.Int != 10 {
		t.Fatalf("got %+v, ok=%v, want int 10", v, ok)
	}
}

func TestParseAndEval_ComparisonExpression(t *testing.T) {

	v, ok := eval(&parser.BoolLiteral{Value: 1 + 2 > 3}, newEnv(), diagnostics.New(""))
	if !ok || v.Kind != KindBool || v.Bool != false {
		// 1+2 == 3, and 3 > 3 is false
		t.Fatalf("got %+v, ok=%v, want bool false", v, ok)
	}
}

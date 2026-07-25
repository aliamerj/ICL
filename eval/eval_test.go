package eval

import (
	"testing"

	"github.com/aliamerj/icl/diagnostics"
	"github.com/aliamerj/icl/parser"
)

func newReporter() *diagnostics.Reporter {
	return diagnostics.New("")
}

// --- Literal evaluation ---

func TestEval_StringLiteral(t *testing.T) {
	v := eval(&parser.StringLiteral{Value: "hello"}, newEnv(), newReporter())
	if v.Kind != KindString || v.Str != "hello" {
		t.Fatalf("got %+v", v)
	}
}

func TestEval_IntLiteral(t *testing.T) {
	v := eval(&parser.IntLiteral{Value: 42}, newEnv(), newReporter())
	if v.Kind != KindInt || v.Int != 42 {
		t.Fatalf("got %+v", v)
	}
}

func TestEval_FloatLiteral(t *testing.T) {
	v := eval(&parser.FloatLiteral{Value: 3.14}, newEnv(), newReporter())
	if v.Kind != KindFloat || v.Float != 3.14 {
		t.Fatalf("got %+v", v)
	}
}

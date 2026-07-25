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
	v, ok := eval(&parser.StringLiteral{Value: "hello"}, newEnv(), newReporter())
	if !ok || v.Kind != KindString || v.Str != "hello" {
		t.Fatalf("got %+v, ok=%v", v, ok)
	}
}

func TestEval_IntLiteral(t *testing.T) {
	v, ok := eval(&parser.IntLiteral{Value: 42}, newEnv(), newReporter())
	if !ok || v.Kind != KindInt || v.Int != 42 {
		t.Fatalf("got %+v, ok=%v", v, ok)
	}
}

func TestEval_FloatLiteral(t *testing.T) {
	v, ok := eval(&parser.FloatLiteral{Value: 3.14}, newEnv(), newReporter())
	if !ok || v.Kind != KindFloat || v.Float != 3.14 {
		t.Fatalf("got %+v, ok=%v", v, ok)
	}
}

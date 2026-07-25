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

func TestEval_ListExpr(t *testing.T) {
	expr := &parser.ListExpr{Elements: []parser.Expression{
		&parser.StringLiteral{Value: "a"},
		&parser.IntLiteral{Value: 1},
	}}
	v, ok := eval(expr, newEnv(), diagnostics.New(""))
	if !ok || v.Kind != KindList || len(v.List) != 2 {
		t.Fatalf("got %+v, ok=%v", v, ok)
	}
	if v.List[0].Str != "a" || v.List[1].Int != 1 {
		t.Errorf("list contents wrong: %+v", v.List)
	}
}

func TestEval_ObjectExpr(t *testing.T) {
	expr := &parser.ObjectExpr{Fields: []*parser.Attribute{
		{Name: &parser.Identifier{Name: "role_arn"}, Value: &parser.StringLiteral{Value: "arn:..."}},
	}}
	v, ok := eval(expr, newEnv(), diagnostics.New(""))
	if !ok || v.Kind != KindObject {
		t.Fatalf("got %+v, ok=%v", v, ok)
	}
	if v.Object["role_arn"].Str != "arn:..." {
		t.Errorf("object contents wrong: %+v", v.Object)
	}
}

func TestValue_NativeRecursesThroughNesting(t *testing.T) {
	v := ObjectValue(map[string]Value{
		"tags": ListValue([]Value{StringValue("a"), StringValue("b")}),
	})
	native, ok := v.Native().(map[string]any)
	if !ok {
		t.Fatalf("Native() = %T, want map[string]any", v.Native())
	}
	tags, ok := native["tags"].([]any)
	if !ok || len(tags) != 2 || tags[0] != "a" {
		t.Errorf("nested Native() conversion wrong: %+v", native)
	}
}

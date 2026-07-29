package eval

import (
	"testing"

	"github.com/aliamerj/icl/parser"
)

// --- Identifier / Environment resolution ---

func TestIdentifier_Found(t *testing.T) {
	env := NewEnv()
	env.set("bucket_name", StringValue("my-bucket"))

	v, ok := eval(&parser.Identifier{Name: "bucket_name"}, env, newReporter())
	if !ok || v.Kind != KindString || v.Str != "my-bucket" {
		t.Fatalf("got %+v, ok=%v", v, ok)
	}
}

func TestIdentifier_Undefined(t *testing.T) {
	reporter := newReporter()
	v, ok := eval(&parser.Identifier{Name: "nope"}, NewEnv(), reporter)
	if ok {
		t.Fatal("expected failure for undefined reference")
	}
	if !reporter.HasErrors() {
		t.Fatal("expected a diagnostic for undefined reference")
	}
	if v.Kind != KindString || v.Str != "" {
		t.Fatalf("unexpected value on failure: %+v", v)
	}
}

func TestIdentifier_ChildEnvironmentSeesParent(t *testing.T) {
	parent := NewEnv()
	parent.set("x", IntValue(1))
	child := parent.child()

	v, ok := eval(&parser.Identifier{Name: "x"}, child, newReporter())
	if !ok || v.Int != 1 {
		t.Fatalf("child scope should see parent value, got %+v, ok=%v", v, ok)
	}
}

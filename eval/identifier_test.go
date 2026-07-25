package eval

import (
	"testing"

	"github.com/aliamerj/icl/parser"
)

// --- Identifier / Environment resolution ---

func TestIdentifier_Found(t *testing.T) {
	env := newEnv()
	env.set("bucket_name", *StringValue("my-bucket"))

	v := eval(&parser.Identifier{Name: "bucket_name"}, env, newReporter())
	if v.Kind != KindString || v.Str != "my-bucket" {
		t.Fatalf("got %+v", v)
	}
}

func TestIdentifier_Undefined(t *testing.T) {
	reporter := newReporter()
	v := eval(&parser.Identifier{Name: "nope"}, newEnv(), reporter)
	if v != nil {
		t.Fatal("expected failure for undefined reference")
	}
	if !reporter.HasErrors() {
		t.Fatal("expected a diagnostic for undefined reference")
	}
}

func TestIdentifier_ChildEnvironmentSeesParent(t *testing.T) {
	parent := newEnv()
	parent.set("x", *IntValue(1))
	child := parent.child()

	v := eval(&parser.Identifier{Name: "x"}, child, newReporter())
	if v.Int != 1 {
		t.Fatalf("child scope should see parent value, got %+v ", v)
	}
}

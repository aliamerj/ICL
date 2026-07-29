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
	v, ok := eval(&parser.StringLiteral{Value: "hello"}, NewEnv(), newReporter())
	if !ok || v.Kind != KindString || v.Str != "hello" {
		t.Fatalf("got %+v, ok=%v", v, ok)
	}
}

func TestEval_IntLiteral(t *testing.T) {
	v, ok := eval(&parser.IntLiteral{Value: 42}, NewEnv(), newReporter())
	if !ok || v.Kind != KindInt || v.Int != 42 {
		t.Fatalf("got %+v, ok=%v", v, ok)
	}
}

func TestEval_FloatLiteral(t *testing.T) {
	v, ok := eval(&parser.FloatLiteral{Value: 3.14}, NewEnv(), newReporter())
	if !ok || v.Kind != KindFloat || v.Float != 3.14 {
		t.Fatalf("got %+v, ok=%v", v, ok)
	}
}

func TestEval_ListExpr(t *testing.T) {
	expr := &parser.ListExpr{Elements: []parser.Expression{
		&parser.StringLiteral{Value: "a"},
		&parser.IntLiteral{Value: 1},
	}}
	v, ok := eval(expr, NewEnv(), diagnostics.New(""))
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
	v, ok := eval(expr, NewEnv(), diagnostics.New(""))
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

func memberExpr(objName, property string) *parser.MemberExpr {
	return &parser.MemberExpr{
		Object:   &parser.Identifier{Name: objName},
		Property: property,
	}
}

func TestEval_ProviderFieldReference_Simple(t *testing.T) {
	env := NewEnv()
	env.Registry.Providers.Add(newProviderCfg("aws", "", map[string]Value{"region": StringValue("eu-west-1")}))

	v, ok := eval(memberExpr("aws", "region"), env, diagnostics.New(""))
	if !ok || v.Kind != KindString || v.Str != "eu-west-1" {
		t.Fatalf("got %+v, ok=%v", v, ok)
	}
}

func TestEval_ProviderFieldReference_NonStringValues(t *testing.T) {
	env := NewEnv()
	env.Registry.Providers.Add(newProviderCfg("aws", "",
		map[string]Value{
			"max_retries":         IntValue(5),
			"allowed_account_ids": ListValue([]Value{StringValue("123")}),
		}))

	v, ok := eval(memberExpr("aws", "max_retries"), env, diagnostics.New(""))
	if !ok || v.Kind != KindInt || v.Int != 5 {
		t.Fatalf("got %+v, ok=%v", v, ok)
	}

	v2, ok := eval(memberExpr("aws", "allowed_account_ids"), env, diagnostics.New(""))
	if !ok || v2.Kind != KindList || v2.List[0].Str != "123" {
		t.Fatalf("got %+v, ok=%v", v2, ok)
	}
}

func TestEval_ProviderFieldReference_AmbiguousWithoutAlias(t *testing.T) {
	env := NewEnv()
	env.Registry.Providers.Add(newProviderCfg("aws", "east", map[string]Value{"region": StringValue("eu-west-1")}))
	env.Registry.Providers.Add(newProviderCfg("aws", "west", map[string]Value{"region": StringValue("us-west-2")}))
	reporter := diagnostics.New("")
	_, ok := eval(memberExpr("aws", "region"), env, reporter)
	if ok {
		t.Fatal("expected ambiguity error for bare 'aws.region' with two aliases")
	}
	if !reporter.HasErrors() {
		t.Fatal("expected a diagnostic")
	}
}

func TestEval_ProviderFieldReference_ExplicitAliasResolvesAmbiguity(t *testing.T) {
	env := NewEnv()
	env.Registry.Providers.Add(newProviderCfg("aws", "east", map[string]Value{"region": StringValue("eu-west-1")}))
  env.Registry.Providers.Add(newProviderCfg("aws", "west", map[string]Value{"region": StringValue("us-west-2")}))

	v, ok := eval(&parser.MemberExpr{Object: memberExpr("aws", "east"), Property: "region"}, env, diagnostics.New(""))
	if !ok || v.Str != "eu-west-1" {
		t.Fatalf("got %+v, ok=%v, want eu-west-1", v, ok)
	}
}

func TestEval_ProviderFieldReference_UndefinedProvider(t *testing.T) {
	env := NewEnv()

	reporter := diagnostics.New("")
	_, ok := eval(memberExpr("gcp", "region"), env, reporter)
	if ok || !reporter.HasErrors() {
		t.Fatal("expected undefined-provider error")
	}
}

func TestEval_ProviderFieldReference_UndefinedField(t *testing.T) {
	env := NewEnv()
	env.Registry.Providers.Add(newProviderCfg("aws", "", map[string]Value{"region": StringValue("eu-west-1")}))

	reporter := diagnostics.New("")
	_, ok := eval(memberExpr("aws", "nonexistent_field"), env, reporter)
	if ok || !reporter.HasErrors() {
		t.Fatal("expected undefined-field error")
	}
}

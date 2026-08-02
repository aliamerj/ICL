package eval

import (
	"strings"
	"testing"

	"github.com/aliamerj/icl/diagnostics"
	"github.com/aliamerj/icl/lexer"
	"github.com/aliamerj/icl/parser"
)

func TestResolveProviderRef_ValidAlias(t *testing.T) {
	env := NewEnv()
	env.Registry.Providers.add(newProviderCfg("aws", "east", nil))

	typ, alias, ok := env.Registry.Providers.resolveRef(memberExpr("aws", "east"), diagnostics.New(""))
	if !ok || typ != "aws" || alias != "east" {
		t.Fatalf("got typ=%q alias=%q ok=%v", typ, alias, ok)
	}
}

func TestResolveProviderRef_BareIdentifierNoAlias(t *testing.T) {
	env := NewEnv()
	env.Registry.Providers.add(newProviderCfg("aws", "", nil))
	typ, alias, ok := env.Registry.Providers.resolveRef(&parser.Identifier{Name: "aws"}, diagnostics.New(""))
	if !ok || typ != "aws" || alias != "" {
		t.Fatalf("got typ=%q alias=%q ok=%v", typ, alias, ok)
	}
}

func TestResolveProviderRef_UndefinedAliasReportsError(t *testing.T) {
	env := NewEnv()
	env.Registry.Providers.add(newProviderCfg("aws", "east", nil))
	reporter := diagnostics.New("")
	_, _, ok := env.Registry.Providers.resolveRef(memberExpr("aws", "wast"), reporter)
	if ok || !reporter.HasErrors() {
		t.Fatal("expected an error for undefined alias 'wast'")
	}
}

func TestValue_RefNativeWrapsInDollarBraces(t *testing.T) {
	v := RefValue("aws_vpc.demo_vpc.id")
	native, ok := v.Native().(string)
	if !ok || native != "${aws_vpc.demo_vpc.id}" {
		t.Errorf("Native() = %v, want ${aws_vpc.demo_vpc.id}", v.Native())
	}
}

func TestEval_VarFieldAccessOnListRejected(t *testing.T) {
	env := NewEnv()
	reporter := diagnostics.New("")

	decl := parseSingleVarDecl(t, `var curr_bucket = {
  tags = [123, "game", true]
}`)
	evalVar(decl, env, reporter)

	expr := &parser.MemberExpr{Object: memberExpr("curr_bucket", "tags"), Property: "name"}
	_, ok := eval(expr, env, reporter)
	if ok || !reporter.HasErrors() {
		t.Fatal("expected an error: 'tags' is a list, not an object")
	}
}

func TestEval_VarNestedFieldAccessValidatesAgainstDefault(t *testing.T) {
	env := NewEnv()
	reporter := diagnostics.New("")

	decl := parseSingleVarDecl(t, `var curr_bucket = {
  obj = { x = 123 }
}`)
	evalVar(decl, env, reporter)

	// valid path — should succeed
	v, ok := eval(&parser.MemberExpr{Object: memberExpr("curr_bucket", "obj"), Property: "x"}, env, reporter)
	if !ok || v.Str != "var.curr_bucket.obj.x" {
		t.Fatalf("got %+v, ok=%v", v, ok)
	}

	// nonexistent field — should fail
	reporter2 := diagnostics.New("")
	_, ok = eval(&parser.MemberExpr{Object: memberExpr("curr_bucket", "obj"), Property: "y"}, env, reporter2)
	if ok || !reporter2.HasErrors() {
		t.Fatal("expected an error: 'obj' has no field 'y'")
	}
}

func TestEval_VarListIndexAccess(t *testing.T) {
	env := NewEnv()
	reporter := diagnostics.New("")

	decl := parseSingleVarDecl(t, `var curr_bucket = {
  tags = [123, "game", true]
}`)
	evalVar(decl, env, reporter)

	expr := &parser.IndexExpr{
		Object: memberExpr("curr_bucket", "tags"),
		Index:  &parser.IntLiteral{Value: 0},
	}
	v, ok := eval(expr, env, reporter)
	if !ok {
		t.Fatalf("expected success, got: %+v", reporter.Diagnostics())
	}
	if v.Kind != KindRef || v.Str != "var.curr_bucket.tags[0]" {
		t.Errorf("got %+v, want KindRef var.curr_bucket.tags[0]", v)
	}
}

func TestEval_VarListIndexOutOfRangeFails(t *testing.T) {
	env := NewEnv()
	reporter := diagnostics.New("")

	decl := parseSingleVarDecl(t, `var curr_bucket = { tags = [123, "game"] }`)
	evalVar(decl, env, reporter)

	expr := &parser.IndexExpr{Object: memberExpr("curr_bucket", "tags"), Index: &parser.IntLiteral{Value: 5}}
	_, ok := eval(expr, env, reporter)
	if ok || !reporter.HasErrors() {
		t.Fatal("expected an index-out-of-range error")
	}
}

func TestParseExpression_IndexAfterMember(t *testing.T) {
	expr := getExprValue(t, "curr_bucket.tags[0]")
	idx, ok := expr.(*parser.IndexExpr)
	if !ok {
		t.Fatalf("got %T, want *ast.IndexExpr", expr)
	}
	if _, ok := idx.Index.(*parser.IntLiteral); !ok {
		t.Errorf("Index = %T, want IntLiteral", idx.Index)
	}
	member, ok := idx.Object.(*parser.MemberExpr)
	if !ok || member.Property != "tags" {
		t.Fatalf("Object = %+v, want MemberExpr(tags)", idx.Object)
	}
}

func TestEval_VarListDynamicIndexFromAnotherVar(t *testing.T) {
	env := NewEnv()
	reporter := diagnostics.New("")

	idxDecl := parseSingleVarDecl(t, `var idx is number = 0`)
	evalVar(idxDecl, env, reporter)

	bucketDecl := parseSingleVarDecl(t, `var curr_bucket = { tags = ["prod", "local"] }`)
	evalVar(bucketDecl, env, reporter)

	expr := &parser.IndexExpr{
		Object: memberExpr("curr_bucket", "tags"),
		Index:  &parser.Identifier{Name: "idx"},
	}
	v, ok := eval(expr, env, reporter)
	if !ok {
		t.Fatalf("expected success, got: %+v", reporter.Diagnostics())
	}
	if v.Kind != KindRef || v.Str != "var.curr_bucket.tags[var.idx]" {
		t.Errorf("got %+v, want KindRef var.curr_bucket.tags[var.idx]", v)
	}
}

func TestEval_VarListDynamicIndexFromResourceAttr(t *testing.T) {
	env := NewEnv()
	reporter := diagnostics.New("")

	resBlock := parseBlock(t, `resource aws_instance as app { ami = "ami-1" }`)
	evalResource(resBlock, env, reporter)

	bucketDecl := parseSingleVarDecl(t, `var curr_bucket = { tags = ["prod", "local"] }`)
	evalVar(bucketDecl, env, reporter)

	expr := &parser.IndexExpr{
		Object: memberExpr("curr_bucket", "tags"),
		Index:  memberExpr("app", "count"),
	}
	v, ok := eval(expr, env, reporter)
	if !ok {
		t.Fatalf("expected success, got: %+v", reporter.Diagnostics())
	}
	if v.Str != "var.curr_bucket.tags[aws_instance.app.count]" {
		t.Errorf("got %q", v.Str)
	}
}

func TestEval_DynamicIndexOnNonListStillFails(t *testing.T) {
	env := NewEnv()
	reporter := diagnostics.New("")

	idxDecl := parseSingleVarDecl(t, `var idx is number = 0`)
	evalVar(idxDecl, env, reporter)
	nameDecl := parseSingleVarDecl(t, `var curr_bucket = { name = "x" }`)
	evalVar(nameDecl, env, reporter)

	expr := &parser.IndexExpr{
		Object: memberExpr("curr_bucket", "name"), // string, not a list
		Index:  &parser.Identifier{Name: "idx"},
	}
	_, ok := eval(expr, env, reporter)
	if ok || !reporter.HasErrors() {
		t.Fatal("expected an error: cannot dynamically index a non-list field")
	}
}

func TestEval_ConstantArithmeticIndexStillWorks(t *testing.T) {
	// Regression: 0+1 folds to a plain KindInt via existing arithmetic
	// eval — must still take the segIndexConst path, not dynamic.
	env := NewEnv()
	reporter := diagnostics.New("")

	decl := parseSingleVarDecl(t, `var curr_bucket = { tags = ["a", "b", "c"] }`)
	evalVar(decl, env, reporter)

	expr := &parser.IndexExpr{
		Object: memberExpr("curr_bucket", "tags"),
		Index:  &parser.BinaryExpr{Left: &parser.IntLiteral{Value: 0}, Operator: "+", Right: &parser.IntLiteral{Value: 1}},
	}
	v, ok := eval(expr, env, reporter)
	if !ok || v.Str != "var.curr_bucket.tags[1]" {
		t.Fatalf("got %+v, ok=%v, want tags[1] (constant-folded)", v, ok)
	}
}

func TestEval_StringIndexStillRejected(t *testing.T) {
	env := NewEnv()
	reporter := diagnostics.New("")

	decl := parseSingleVarDecl(t, `var curr_bucket = { tags = ["a", "b"] }`)
	evalVar(decl, env, reporter)

	expr := &parser.IndexExpr{
		Object: memberExpr("curr_bucket", "tags"),
		Index:  &parser.StringLiteral{Value: "0"},
	}
	_, ok := eval(expr, env, reporter)
	if ok || !reporter.HasErrors() {
		t.Fatal("expected an error for a string index")
	}
}

func TestEval_ProviderConstIndexResolvesEagerly(t *testing.T) {
	env := NewEnv()
	env.Registry.Providers.add(&ProviderConfig{Name: "aws", Extra: map[string]Value{
		"allowed_account_ids": ListValue([]Value{StringValue("111"), StringValue("222")}),
	}})
	reporter := diagnostics.New("")

	expr := &parser.IndexExpr{
		Object: memberExpr("aws", "allowed_account_ids"),
		Index:  &parser.IntLiteral{Value: 1},
	}
	v, ok := eval(expr, env, reporter)
	if !ok {
		t.Fatalf("expected success, got: %+v", reporter.Diagnostics())
	}
	if v.Kind != KindString || v.Str != "222" {
		t.Fatalf("got %+v, want literal string 222 (NOT a RefValue)", v)
	}
}

func TestEval_ProviderNestedFieldEagerly(t *testing.T) {
	env := NewEnv()
	env.Registry.Providers.add(&ProviderConfig{Name: "aws", Extra: map[string]Value{
		"assume_role": ObjectValue(map[string]Value{"role_arn": StringValue("arn:aws:iam::123:role/deploy")}),
	}})
	reporter := diagnostics.New("")

	expr := &parser.MemberExpr{Object: memberExpr("aws", "assume_role"), Property: "role_arn"}
	v, ok := eval(expr, env, reporter)
	if !ok || v.Kind != KindString || v.Str != "arn:aws:iam::123:role/deploy" {
		t.Fatalf("got %+v, ok=%v", v, ok)
	}
}

func TestEval_ProviderDynamicIndexRejected(t *testing.T) {
	env := NewEnv()

	env.Registry.Providers.add(&ProviderConfig{Name: "aws", Extra: map[string]Value{
		"allowed_account_ids": ListValue([]Value{StringValue("111")}),
	}})
	reporter := diagnostics.New("")

	idxDecl := parseSingleVarDecl(t, `var idx is number = 0`)
	evalVar(idxDecl, env, reporter)

	expr := &parser.IndexExpr{
		Object: memberExpr("aws", "allowed_account_ids"),
		Index:  &parser.Identifier{Name: "idx"},
	}
	_, ok := eval(expr, env, reporter)
	if ok || !reporter.HasErrors() {
		t.Fatal("expected an error: dynamic index into a provider field must be rejected")
	}
}

func TestEval_ProviderIndexOutOfRangeStillCaught(t *testing.T) {
	env := NewEnv()

	env.Registry.Providers.add(&ProviderConfig{Name: "aws", Extra: map[string]Value{
		"allowed_account_ids": ListValue([]Value{StringValue("111")}),
	}})
	reporter := diagnostics.New("")

	expr := &parser.IndexExpr{
		Object: memberExpr("aws", "allowed_account_ids"),
		Index:  &parser.IntLiteral{Value: 5},
	}
	_, ok := eval(expr, env, reporter)
	if ok || !reporter.HasErrors() {
		t.Fatal("expected index-out-of-range error")
	}
}

func TestEval_UndefinedReferenceHintsAtForwardDeclaration(t *testing.T) {
	env := NewEnv()
	env.SetForwardLookup(func(name string) (string, bool) {
		if name == "curr_bucket_name" {
			return "var", true
		}
		return "", false
	})
	reporter := diagnostics.New("")

	_, ok := eval(&parser.Identifier{Name: "curr_bucket_name"}, env, reporter)
	if ok {
		t.Fatal("expected failure — reference is still unresolved, hint doesn't change that")
	}
	diags := reporter.Diagnostics()
	if len(diags) == 0 || !strings.Contains(diags[0].Help, "declared later") {
		t.Fatalf("expected a forward-declaration hint, got: %+v", diags)
	}
}

func TestEval_UndefinedReferenceNoHintWhenTrulyUndefined(t *testing.T) {
	env := NewEnv()
	env.SetForwardLookup(func(name string) (string, bool) { return "", false })
	reporter := diagnostics.New("")

	_, ok := eval(&parser.Identifier{Name: "totally_made_up"}, env, reporter)
	if ok {
		t.Fatal("expected failure")
	}
	diags := reporter.Diagnostics()
	if len(diags) == 0 || strings.Contains(diags[0].Help, "declared later") {
		t.Fatalf("should NOT get a forward-declaration hint for a name that doesn't exist at all: %+v", diags)
	}
}

func TestEvalReferenceChain_UndefinedReferenceHintsAtForwardDeclaration(t *testing.T) {
	env := NewEnv()
	env.SetForwardLookup(func(name string) (string, bool) {
		if name == "demo_vpc" {
			return "resource", true
		}
		return "", false
	})
	reporter := diagnostics.New("")

	expr := &parser.MemberExpr{Object: &parser.Identifier{Name: "demo_vpc"}, Property: "id"}
	_, ok := eval(expr, env, reporter)
	if ok {
		t.Fatal("expected failure — hint doesn't make it resolve")
	}
	diags := reporter.Diagnostics()
	if len(diags) == 0 || !strings.Contains(diags[0].Help, "declared later") {
		t.Fatalf("expected a forward-declaration hint, got: %+v", diags)
	}
}

func getExprValue(t *testing.T, src string) parser.Expression {
	t.Helper()
	full := "provider aws {\n  x = " + src + "\n}"
	prog, reporter := parse(t, full)
	if reporter.HasErrors() {
		t.Fatalf("unexpected parse errors for %q: %+v", src, reporter.Diagnostics())
	}
	block := prog.Statements[0].(*parser.Block)
	attr := block.Body.Statements[0].(*parser.Attribute)
	return attr.Value
}

func parse(t *testing.T, source string) (*parser.Program, *diagnostics.Reporter) {
	t.Helper()
	scan := lexer.New(source, diagnostics.New(source))
	reporter := diagnostics.New(source)
	p := parser.New(scan.Tokens(), reporter)
	prog := p.ParseProgram()
	return prog, reporter
}

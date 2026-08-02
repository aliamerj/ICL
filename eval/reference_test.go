package eval

import (
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

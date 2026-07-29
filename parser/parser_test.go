package parser

import (
	"testing"
	"time"

	"github.com/aliamerj/icl/diagnostics"
	"github.com/aliamerj/icl/lexer"
	"github.com/aliamerj/icl/tokens"
)

func parse(t *testing.T, source string) (*Program, *diagnostics.Reporter) {
	t.Helper()
	scan := lexer.New(source, diagnostics.New(source))
	reporter := diagnostics.New(source)
	p := New(scan.Tokens(), reporter)
	prog := p.ParseProgram()
	return prog, reporter
}

func TestParseProvider_HappyPath(t *testing.T) {
	src := `
provider aws {
  source  = "hashicorp/aws"
  version = "5.37.0"
}
`
	prog, reporter := parse(t, src)

	if reporter.HasErrors() {
		t.Fatalf("unexpected errors: %+v", reporter.Diagnostics())
	}
	if len(prog.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(prog.Statements))
	}

	block, ok := prog.Statements[0].(*Block)
	if !ok {
		t.Fatalf("expected *Block, got %T", prog.Statements[0])
	}
	if block.Keyword != tokens.PROVIDER {
		t.Errorf("Keyword = %q, want provider", block.Keyword)
	}
	if len(block.Labels) != 1 || block.Labels[0].Name != "aws" {
		t.Fatalf("Labels = %+v, want [aws]", block.Labels)
	}
	if len(block.Body.Statements) != 2 {
		t.Fatalf("expected 2 attributes, got %d", len(block.Body.Statements))
	}

	attr0, ok := block.Body.Statements[0].(*Attribute)
	if !ok {
		t.Fatalf("expected *Attribute, got %T", block.Body.Statements[0])
	}
	if attr0.Name.Name != "source" {
		t.Errorf("attr0.Name = %q, want source", attr0.Name.Name)
	}
	sv, ok := attr0.Value.(*StringLiteral)
	if !ok || sv.Value != "hashicorp/aws" {
		t.Errorf("attr0.Value = %+v, want hashicorp/aws", attr0.Value)
	}

	attr1 := block.Body.Statements[1].(*Attribute)
	if attr1.Name.Name != "version" {
		t.Errorf("attr1.Name = %q, want version", attr1.Name.Name)
	}
}

func TestParseProvider_EmptyBody(t *testing.T) {
	src := `provider aws {}`
	prog, reporter := parse(t, src)

	if reporter.HasErrors() {
		t.Fatalf("unexpected errors: %+v", reporter.Diagnostics())
	}
	if len(prog.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(prog.Statements))
	}
	block := prog.Statements[0].(*Block)
	if len(block.Body.Statements) != 0 {
		t.Errorf("expected 0 attributes, got %d", len(block.Body.Statements))
	}
}

func TestParseProvider_MissingLabel(t *testing.T) {
	src := `provider {
  region = "eu-west-1"
}`
	prog, reporter := parse(t, src)

	if !reporter.HasErrors() {
		t.Fatal("expected an error for missing label, got none")
	}
	if len(prog.Statements) != 0 {
		t.Errorf("expected 0 statements after failed block, got %d", len(prog.Statements))
	}
}

func TestParseProvider_MissingOpenBrace(t *testing.T) {
	src := `provider aws
  region = "eu-west-1"
}`
	_, reporter := parse(t, src)

	if !reporter.HasErrors() {
		t.Fatal("expected an error for missing '{', got none")
	}
}

func TestParseAttribute_MissingEquals(t *testing.T) {
	src := `provider aws {
  region "eu-west-1"
}`
	_, reporter := parse(t, src)

	if !reporter.HasErrors() {
		t.Fatal("expected an error for missing '=', got none")
	}
}

func TestParseAttribute_MissingValue(t *testing.T) {
	src := `provider aws {
  region =
}`
	_, reporter := parse(t, src)

	if !reporter.HasErrors() {
		t.Fatal("expected an error for missing value, got none")
	}
}

func TestParseProvider_UnclosedBlock(t *testing.T) {
	src := `provider aws {
  region = "eu-west-1"
`
	_, reporter := parse(t, src)

	if !reporter.HasErrors() {
		t.Fatal("expected an error for unclosed block, got none")
	}
}

func TestParseProgram_RecoversAfterBadTopLevelToken(t *testing.T) {
	// A stray identifier at top level should error, but parsing
	// should recover and still pick up the valid provider block after it.
	src := `
garbage

provider aws {
  region = "eu-west-1"
}
`
	prog, reporter := parse(t, src)

	if !reporter.HasErrors() {
		t.Fatal("expected an error for stray top-level token, got none")
	}
	if len(prog.Statements) != 1 {
		t.Fatalf("expected recovery to still parse 1 valid statement, got %d :: %+v", len(prog.Statements), prog.Statements)
	}
	block := prog.Statements[0].(*Block)
	if block.Keyword != tokens.PROVIDER || len(block.Labels) != 1 || block.Labels[0].Name != "aws" {
		t.Errorf("recovered block is wrong: %+v", block)
	}
}

func TestParseProgram_RecoversAfterBrokenBlock(t *testing.T) {
	// First provider block is missing its value; second is well-formed.
	// synchronize() should skip past the first block's '}' and let the
	// second one parse cleanly.
	src := `
provider aws {
  region =
}

provider gcp {
  region = "us-central1"
}
`
	prog, reporter := parse(t, src)

	if !reporter.HasErrors() {
		t.Fatal("expected an error from the first broken block, got none")
	}
	if len(prog.Statements) != 1 {
		t.Fatalf("expected exactly 1 recovered statement, got %d", len(prog.Statements))
	}
	block := prog.Statements[0].(*Block)
	if len(block.Labels) != 1 || block.Labels[0].Name != "gcp" {
		t.Errorf("expected recovered block to be gcp, got %+v", block.Labels)
	}
}

func TestParseProvider_Ranges(t *testing.T) {
	src := `provider aws {
  region = "eu-west-1"
}`
	prog, reporter := parse(t, src)
	if reporter.HasErrors() {
		t.Fatalf("unexpected errors: %+v", reporter.Diagnostics())
	}

	block := prog.Statements[0].(*Block)
	if block.Rng.Start.Offset != 0 {
		t.Errorf("block start offset = %d, want 0", block.Rng.Start.Offset)
	}
	if block.Rng.End.Offset <= block.Rng.Start.Offset {
		t.Errorf("block end offset (%d) should be after start (%d)",
			block.Rng.End.Offset, block.Rng.Start.Offset)
	}

	attr := block.Body.Statements[0].(*Attribute)
	if attr.Name.Rng.Start.Offset == 0 {
		t.Error("attribute name offset wasn't set (still zero value)")
	}
}

func TestParseAttribute_IntLiteral(t *testing.T) {
	src := `provider aws {
  someNumber = 5
}`
	prog, reporter := parse(t, src)
	if reporter.HasErrors() {
		t.Fatalf("unexpected errors: %+v", reporter.Diagnostics())
	}

	block := prog.Statements[0].(*Block)
	attr := block.Body.Statements[0].(*Attribute)

	if attr.Name.Name != "someNumber" {
		t.Errorf("attr.Name = %q, want someNumber", attr.Name.Name)
	}

	lit, ok := attr.Value.(*IntLiteral)
	if !ok {
		t.Fatalf("attr.Value type = %T, want *IntLiteral", attr.Value)
	}
	if lit.Value != 5 {
		t.Errorf("lit.Value = %d, want 5", lit.Value)
	}
}

func TestParseAttribute_FloatLiteral(t *testing.T) {
	src := `provider aws {
  someFloat = 12.6
}`
	prog, reporter := parse(t, src)
	if reporter.HasErrors() {
		t.Fatalf("unexpected errors: %+v", reporter.Diagnostics())
	}

	block := prog.Statements[0].(*Block)
	attr := block.Body.Statements[0].(*Attribute)

	lit, ok := attr.Value.(*FloatLiteral)
	if !ok {
		t.Fatalf("attr.Value type = %T, want *FloatLiteral", attr.Value)
	}
	if lit.Value != 12.6 {
		t.Errorf("lit.Value = %v, want 12.6", lit.Value)
	}
}

func TestParseAttribute_MixedLiteralTypes(t *testing.T) {
	// Confirms string/int/float can all coexist in the same block,
	// each producing the correct concrete AST type.
	src := `provider aws {
  source     = "hashicorp/aws"
  version    = "5.37.0"
  someNumber = 5
  someFloat  = 12.6
}`
	prog, reporter := parse(t, src)
	if reporter.HasErrors() {
		t.Fatalf("unexpected errors: %+v", reporter.Diagnostics())
	}

	block := prog.Statements[0].(*Block)
	if len(block.Body.Statements) != 4 {
		t.Fatalf("expected 4 attributes, got %d", len(block.Body.Statements))
	}

	wantTypes := []struct {
		name string
		want any
	}{
		{"source", &StringLiteral{}},
		{"version", &StringLiteral{}},
		{"someNumber", &IntLiteral{}},
		{"someFloat", &FloatLiteral{}},
	}

	for i, want := range wantTypes {
		attr := block.Body.Statements[i].(*Attribute)
		if attr.Name.Name != want.name {
			t.Errorf("attribute %d: name = %q, want %q", i, attr.Name.Name, want.name)
		}

		gotType := attr.Value
		switch want.want.(type) {
		case *StringLiteral:
			if _, ok := gotType.(*StringLiteral); !ok {
				t.Errorf("attribute %d (%s): type = %T, want *StringLiteral", i, want.name, gotType)
			}
		case *IntLiteral:
			if _, ok := gotType.(*IntLiteral); !ok {
				t.Errorf("attribute %d (%s): type = %T, want *IntLiteral", i, want.name, gotType)
			}
		case *FloatLiteral:
			if _, ok := gotType.(*FloatLiteral); !ok {
				t.Errorf("attribute %d (%s): type = %T, want *FloatLiteral", i, want.name, gotType)
			}
		}
	}
}

func TestParseAttribute_NegativeOrExponentNumbersNotYetSupported(t *testing.T) {
	// Documents current behavior: parseExpression only knows STRING,
	// NUMBER_INT, NUMBER_FLOAT. Anything else (like a leading '-')
	// should fail cleanly with a diagnostic, not panic or silently
	// produce a wrong value. Revisit this test once unary minus
	// or exponents are added to the grammar.
	src := `provider aws {
  someNumber = -5
}`
	_, reporter := parse(t, src)

	if !reporter.HasErrors() {
		t.Fatal("expected an error, since unary '-' isn't supported yet")
	}
}

func TestParseAttribute_UnsupportedValueTokenReportsError(t *testing.T) {
	src := `provider aws {
  region = {
}`
	_, reporter := parse(t, src)

	if !reporter.HasErrors() {
		t.Fatal("expected an error for an unsupported value token")
	}
}

func TestParseAttribute_IntLiteral_Range(t *testing.T) {
	src := `provider aws {
  port = 8080
}`
	prog, reporter := parse(t, src)
	if reporter.HasErrors() {
		t.Fatalf("unexpected errors: %+v", reporter.Diagnostics())
	}

	block := prog.Statements[0].(*Block)
	attr := block.Body.Statements[0].(*Attribute)
	lit := attr.Value.(*IntLiteral)

	if lit.Rng.Start.Offset == 0 {
		t.Error("IntLiteral range offset wasn't set (still zero value)")
	}
	if attr.Rng.End != lit.Rng.End {
		t.Errorf("Attribute.Rng.End = %+v, want it to match value's end %+v", attr.Rng.End, lit.Rng.End)
	}
}

func TestParseProgram_RangeCoversWholeFile(t *testing.T) {
	src := `provider aws {
  region = "eu-west-1"
}`
	prog, reporter := parse(t, src)
	if reporter.HasErrors() {
		t.Fatalf("unexpected errors: %+v", reporter.Diagnostics())
	}
	if prog.Rng.Start.Offset != 0 {
		t.Errorf("Program.Rng.Start.Offset = %d, want 0", prog.Rng.Start.Offset)
	}
	if prog.Rng.End.Offset != len(src) {
		t.Errorf("Program.Rng.End.Offset = %d, want %d", prog.Rng.End.Offset, len(src))
	}
}

// getExprValue is a helper: parses a single provider block with one
// attribute, and returns that attribute's parsed Expression.
func getExprValue(t *testing.T, src string) Expression {
	t.Helper()
	full := "provider aws {\n  x = " + src + "\n}"
	prog, reporter := parse(t, full)
	if reporter.HasErrors() {
		t.Fatalf("unexpected parse errors for %q: %+v", src, reporter.Diagnostics())
	}
	block := prog.Statements[0].(*Block)
	attr := block.Body.Statements[0].(*Attribute)
	return attr.Value
}

func TestParseExpression_SimpleAddition(t *testing.T) {
	expr := getExprValue(t, "1 + 2")

	bin, ok := expr.(*BinaryExpr)
	if !ok {
		t.Fatalf("got %T, want *BinaryExpr", expr)
	}
	if bin.Operator != "+" {
		t.Errorf("Operator = %q, want +", bin.Operator)
	}
	left, ok := bin.Left.(*IntLiteral)
	if !ok || left.Value != 1 {
		t.Errorf("Left = %+v, want IntLiteral(1)", bin.Left)
	}
	right, ok := bin.Right.(*IntLiteral)
	if !ok || right.Value != 2 {
		t.Errorf("Right = %+v, want IntLiteral(2)", bin.Right)
	}
}

func TestParseExpression_MultiplicationBindsTighterThanAddition(t *testing.T) {
	// 1 + 2 * 3  must parse as  1 + (2 * 3), i.e. top-level operator is '+'
	expr := getExprValue(t, "1 + 2 * 3")

	top, ok := expr.(*BinaryExpr)
	if !ok || top.Operator != "+" {
		t.Fatalf("top-level = %+v, want '+' at the top", expr)
	}
	if _, ok := top.Left.(*IntLiteral); !ok {
		t.Errorf("Left = %T, want IntLiteral(1)", top.Left)
	}
	right, ok := top.Right.(*BinaryExpr)
	if !ok || right.Operator != "*" {
		t.Fatalf("Right = %+v, want a nested '*' expression", top.Right)
	}
}

func TestParseExpression_LeftAssociativity(t *testing.T) {
	// 10 - 3 - 2  must parse as  (10 - 3) - 2, not 10 - (3 - 2)
	expr := getExprValue(t, "10 - 3 - 2")

	top, ok := expr.(*BinaryExpr)
	if !ok || top.Operator != "-" {
		t.Fatalf("top-level = %+v, want '-' at the top", expr)
	}
	left, ok := top.Left.(*BinaryExpr)
	if !ok || left.Operator != "-" {
		t.Fatalf("Left = %+v, want a nested '-' expression (10-3)", top.Left)
	}
	rightLit, ok := top.Right.(*IntLiteral)
	if !ok || rightLit.Value != 2 {
		t.Errorf("Right = %+v, want IntLiteral(2)", top.Right)
	}
}

func TestParseExpression_Parentheses(t *testing.T) {
	// (1 + 2) * 4  must parse with '*' at the top, '+' nested on the left
	expr := getExprValue(t, "(1 + 2) * 4")

	top, ok := expr.(*BinaryExpr)
	if !ok || top.Operator != "*" {
		t.Fatalf("top-level = %+v, want '*' at the top", expr)
	}
	left, ok := top.Left.(*BinaryExpr)
	if !ok || left.Operator != "+" {
		t.Fatalf("Left = %+v, want a nested '+' expression", top.Left)
	}
}

func TestParseExpression_ComparisonBindsLooserThanArithmetic(t *testing.T) {
	// 1 + 2 > 3  must parse as  (1 + 2) > 3
	expr := getExprValue(t, "1 + 2 > 3")

	top, ok := expr.(*BinaryExpr)
	if !ok || top.Operator != ">" {
		t.Fatalf("top-level = %+v, want '>' at the top", expr)
	}
	left, ok := top.Left.(*BinaryExpr)
	if !ok || left.Operator != "+" {
		t.Fatalf("Left = %+v, want a nested '+' expression", top.Left)
	}
}

func TestParseExpression_Identifier(t *testing.T) {
	expr := getExprValue(t, "bucket_name")

	id, ok := expr.(*Identifier)
	if !ok || id.Name != "bucket_name" {
		t.Fatalf("got %+v, want Identifier(bucket_name)", expr)
	}
}

func TestParseExpression_IdentifierInArithmetic(t *testing.T) {
	expr := getExprValue(t, "count + 1")

	bin, ok := expr.(*BinaryExpr)
	if !ok || bin.Operator != "+" {
		t.Fatalf("got %+v, want '+' expression", expr)
	}
	if _, ok := bin.Left.(*Identifier); !ok {
		t.Errorf("Left = %T, want *Identifier", bin.Left)
	}
}

func TestParseExpression_StillWorksForPlainLiterals(t *testing.T) {
	// Regression check: existing single-literal attributes must still
	// parse exactly as before, now that parseExpression routes through
	// precedence climbing instead of returning a literal directly.
	strExpr := getExprValue(t, `"hello"`)
	if s, ok := strExpr.(*StringLiteral); !ok || s.Value != "hello" {
		t.Errorf("got %+v, want StringLiteral(hello)", strExpr)
	}

	floatExpr := getExprValue(t, "12.6")
	if f, ok := floatExpr.(*FloatLiteral); !ok || f.Value != 12.6 {
		t.Errorf("got %+v, want FloatLiteral(12.6)", floatExpr)
	}
}

func TestParseExpression_UnclosedParenReportsError(t *testing.T) {
	full := "provider aws {\n  x = (1 + 2\n}"
	_, reporter := parse(t, full)
	if !reporter.HasErrors() {
		t.Fatal("expected an error for unclosed parenthesis")
	}
}

func TestParseExpression_MissingRightOperandReportsErrorAndDoesNotHang(t *testing.T) {
	full := "provider aws {\n  x = 1 +\n}"
	done := make(chan struct{})
	var reporter *diagnostics.Reporter

	go func() {
		_, reporter = parse(t, full)
		close(done)
	}()

	select {
	case <-done:
		if !reporter.HasErrors() {
			t.Fatal("expected an error for a missing right operand")
		}
	case <-timeoutChan():
		t.Fatal("parser hung on '1 +' with no right operand")
	}
}

func timeoutChan() <-chan struct{} {
	ch := make(chan struct{})
	go func() {
		time.Sleep(2 * time.Second)
		close(ch)
	}()
	return ch
}

func TestParsePrimary_MalformedLiteralDoesNotPanic(t *testing.T) {
	tokens := []lexer.Token{{Type: tokens.STRING, Lexeme: "\"broken\"", Literal: 123, Line: 1, Offset: 0}, {Type: tokens.EOF, Line: 1, Offset: 9}}
	reporter := diagnostics.New("")
	p := New(tokens, reporter)
	prog := p.ParseProgram()

	if prog == nil {
		t.Fatal("expected a program, got nil")
	}
	if !reporter.HasErrors() {
		t.Fatal("expected malformed literal to report an error")
	}
}

func TestParseResource_MissingNameIsAHardError(t *testing.T) {
	// requireName=true for resource — unlike provider, omitting `as name`
	// must fail, since an unnamed resource can never be referenced.
	src := `
resource aws_instance {
  ami = "ami-123456"
}
`
	prog, reporter := parse(t, src)

	if !reporter.HasErrors() {
		t.Fatal("expected an error for a resource with no name")
	}
	if len(prog.Statements) != 0 {
		t.Errorf("expected 0 statements after failed block, got %d", len(prog.Statements))
	}
}

func TestParseResource_MissingLabel(t *testing.T) {
	src := `
resource as app_server {
  ami = "ami-123456"
}
`
	_, reporter := parse(t, src)
	if !reporter.HasErrors() {
		t.Fatal("expected an error for a resource with no type label")
	}
}

func TestParseResource_MissingOpenBrace(t *testing.T) {
	src := `
resource aws_instance as app_server
  ami = "ami-123456"
}
`
	_, reporter := parse(t, src)
	if !reporter.HasErrors() {
		t.Fatal("expected an error for missing '{'")
	}
}

func TestParseResource_UnclosedBlock(t *testing.T) {
	src := `
resource aws_instance as app_server {
  ami = "ami-123456"
`
	_, reporter := parse(t, src)
	if !reporter.HasErrors() {
		t.Fatal("expected an error for an unclosed block")
	}
}

func TestParseResource_EmptyBody(t *testing.T) {
	src := `resource aws_instance as app_server {}`
	prog, reporter := parse(t, src)
	if reporter.HasErrors() {
		t.Fatalf("unexpected errors: %+v", reporter.Diagnostics())
	}
	block := prog.Statements[0].(*Block)
	if len(block.Body.Statements) != 0 {
		t.Errorf("expected 0 attributes, got %d", len(block.Body.Statements))
	}
}

func TestParseResource_MemberExprAttributeValue(t *testing.T) {
	// The exact shape resource blocks exist for: referencing another
	// declared value (vpc_id = demo_vpc.id).
	src := `
resource aws_subnet as public_subnet {
  vpc_id     = demo_vpc.id
  cidr_block = "10.0.0.0/24"
}
`
	prog, reporter := parse(t, src)
	if reporter.HasErrors() {
		t.Fatalf("unexpected errors: %+v", reporter.Diagnostics())
	}

	block := prog.Statements[0].(*Block)
	attr := block.Body.Statements[0].(*Attribute)
	member, ok := attr.Value.(*MemberExpr)
	if !ok {
		t.Fatalf("attr.Value type = %T, want *ast.MemberExpr", attr.Value)
	}
	base, ok := member.Object.(*Identifier)
	if !ok || base.Name != "demo_vpc" || member.Property != "id" {
		t.Errorf("MemberExpr = %+v, want demo_vpc.id", member)
	}
}

func TestParseResource_ProviderMetaAttribute(t *testing.T) {
	// provider = aws.east — parses as an ordinary attribute at this stage;
	// EvalResource is responsible for pulling it out specially later.
	src := `
resource aws_instance as app_server {
  ami      = "ami-123456"
  provider = aws.east
}
`
	prog, reporter := parse(t, src)
	if reporter.HasErrors() {
		t.Fatalf("unexpected errors: %+v", reporter.Diagnostics())
	}
	block := prog.Statements[0].(*Block)
	attr := block.Body.Statements[1].(*Attribute)
	if attr.Name.Name != "provider" {
		t.Fatalf("attr.Name = %q, want provider", attr.Name.Name)
	}
	if _, ok := attr.Value.(*MemberExpr); !ok {
		t.Errorf("attr.Value type = %T, want *ast.MemberExpr", attr.Value)
	}
}

func TestParseResource_Range(t *testing.T) {
	src := `resource aws_instance as app_server {
  ami = "ami-123456"
}`
	prog, reporter := parse(t, src)
	if reporter.HasErrors() {
		t.Fatalf("unexpected errors: %+v", reporter.Diagnostics())
	}
	block := prog.Statements[0].(*Block)
	if block.Rng.Start.Offset != 0 {
		t.Errorf("Rng.Start.Offset = %d, want 0", block.Rng.Start.Offset)
	}
	if block.Rng.End.Offset <= block.Rng.Start.Offset {
		t.Errorf("Rng.End.Offset (%d) should be after Start (%d)", block.Rng.End.Offset, block.Rng.Start.Offset)
	}
}

func TestParseProvider_StillOptionalNameAfterSharedParseBlock(t *testing.T) {
	// Regression: extracting parseBlock(keyword, requireName) must not
	// break provider's existing "name is optional" behavior.
	src := `provider aws {
  region = "eu-west-1"
}`
	prog, reporter := parse(t, src)
	if reporter.HasErrors() {
		t.Fatalf("unexpected errors: %+v", reporter.Diagnostics())
	}
	block := prog.Statements[0].(*Block)
	if block.Name != nil {
		t.Errorf("Name = %+v, want nil (provider name should still be optional)", block.Name)
	}
}

func TestParseProvider_StillSupportsOptionalAliasName(t *testing.T) {
	src := `provider aws as east {
  region = "us-east-1"
}`
	prog, reporter := parse(t, src)
	if reporter.HasErrors() {
		t.Fatalf("unexpected errors: %+v", reporter.Diagnostics())
	}
	block := prog.Statements[0].(*Block)
	if block.Name == nil || block.Name.Name != "east" {
		t.Errorf("Name = %+v, want east", block.Name)
	}
}

func TestParseProgram_RecoversAcrossDifferentKeywords(t *testing.T) {
	src := `
provider aws {
  region =
}

resource aws_instance as app_server {
  ami = "ami-123"
}
`
	prog, reporter := parse(t, src)

	if !reporter.HasErrors() {
		t.Fatal("expected an error from the broken provider block")
	}
	if len(prog.Statements) != 1 {
		t.Fatalf("expected 1 recovered statement, got %d", len(prog.Statements))
	}
	block := prog.Statements[0].(*Block)
	if block.Keyword != tokens.RESOURCE || block.Name == nil || block.Name.Name != "app_server" {
		t.Errorf("recovered block is wrong: %+v", block)
	}
}

package parser

import (
	"testing"

	"github.com/aliamerj/icl/diagnostics"
	"github.com/aliamerj/icl/lexer"
)

func parseExpr(t *testing.T, input string) Expression {
	t.Helper()

	l := lexer.New(input, diagnostics.New(input))
	p := New(l.Tokens(), diagnostics.New(input))

	expr := p.parseExpression()
	if expr == nil {
		t.Fatalf("expected expression, got nil")
	}

	if len(p.reporter.Diagnostics()) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", p.reporter.Diagnostics())
	}

	return expr
}

// Literals
func TestParseIntLiteral(t *testing.T) {
	expr := parseExpr(t, "42")

	lit, ok := expr.(*IntLiteral)
	if !ok {
		t.Fatalf("expected *IntLiteral, got %T", expr)
	}

	if lit.Value != 42 {
		t.Fatalf("expected 42, got %d", lit.Value)
	}
}

func TestParseFloatLiteral(t *testing.T) {
	expr := parseExpr(t, "3.14")

	lit := expr.(*FloatLiteral)

	if lit.Value != 3.14 {
		t.Fatalf("expected 3.14, got %f", lit.Value)
	}
}

func TestParseTrueLiteral(t *testing.T) {
	expr := parseExpr(t, "true")

	lit := expr.(*BoolLiteral)

	if !lit.Value {
		t.Fatal("expected true")
	}
}

func TestParseFalseLiteral(t *testing.T) {
	expr := parseExpr(t, "false")

	lit := expr.(*BoolLiteral)

	if lit.Value {
		t.Fatal("expected false")
	}
}

// Identifier
func TestParseIdentifier(t *testing.T) {
	expr := parseExpr(t, "name")

	id, ok := expr.(*Identifier)
	if !ok {
		t.Fatalf("expected identifier, got %T", expr)
	}

	if id.Name != "name" {
		t.Fatalf("expected name, got %q", id.Name)
	}
}

// Parentheses
func TestParseParenthesizedExpression(t *testing.T) {
	expr := parseExpr(t, "(42)")

	lit, ok := expr.(*IntLiteral)
	if !ok {
		t.Fatalf("expected int literal, got %T", expr)
	}

	if lit.Value != 42 {
		t.Fatal("wrong value")
	}
}

// Addition
func TestParseAddition(t *testing.T) {
	expr := parseExpr(t, "1 + 2")

	bin, ok := expr.(*BinaryExpr)
	if !ok {
		t.Fatalf("expected binary expr")
	}

	if bin.Operator != "+" {
		t.Fatalf("expected +")
	}

	left := bin.Left.(*IntLiteral)
	right := bin.Right.(*IntLiteral)

	if left.Value != 1 {
		t.Fatal("wrong left")
	}

	if right.Value != 2 {
		t.Fatal("wrong right")
	}
}

// Multiplication
func TestParseMultiplication(t *testing.T) {
	expr := parseExpr(t, "2 * 3")

	bin := expr.(*BinaryExpr)

	if bin.Operator != "*" {
		t.Fatal("expected *")
	}
}

// Operator precedence
func TestOperatorPrecedence(t *testing.T) {
	expr := parseExpr(t, "1 + 2 * 3")

	add := expr.(*BinaryExpr)

	if add.Operator != "+" {
		t.Fatalf("expected +")
	}

	left := add.Left.(*IntLiteral)

	if left.Value != 1 {
		t.Fatal("wrong left")
	}

	mul := add.Right.(*BinaryExpr)

	if mul.Operator != "*" {
		t.Fatalf("expected *")
	}

	if mul.Left.(*IntLiteral).Value != 2 {
		t.Fatal()
	}

	if mul.Right.(*IntLiteral).Value != 3 {
		t.Fatal()
	}
}

// Parentheses override precedence
func TestParenthesesOverridePrecedence(t *testing.T) {
	expr := parseExpr(t, "(1 + 2) * 3")

	mul := expr.(*BinaryExpr)

	if mul.Operator != "*" {
		t.Fatal()
	}

	add := mul.Left.(*BinaryExpr)

	if add.Operator != "+" {
		t.Fatal()
	}
}

// Left associativity
func TestLeftAssociativity(t *testing.T) {
	expr := parseExpr(t, "1 - 2 - 3")

	root := expr.(*BinaryExpr)

	if root.Operator != "-" {
		t.Fatal()
	}

	left := root.Left.(*BinaryExpr)

	if left.Operator != "-" {
		t.Fatal()
	}
}

// Comparison
func TestComparison(t *testing.T) {
	expr := parseExpr(t, "1 < 2")

	bin := expr.(*BinaryExpr)

	if bin.Operator != "<" {
		t.Fatal()
	}
}

// Mixed precedence
func TestComparisonPrecedence(t *testing.T) {
	expr := parseExpr(t, "1 + 2 < 3 * 4")

	root := expr.(*BinaryExpr)

	if root.Operator != "<" {
		t.Fatalf("expected <")
	}

	if root.Left.(*BinaryExpr).Operator != "+" {
		t.Fatal()
	}

	if root.Right.(*BinaryExpr).Operator != "*" {
		t.Fatal()
	}
}

// Invalid input
func TestUnexpectedToken(t *testing.T) {
	l := lexer.New(")", diagnostics.New(")"))
	p := New(l.Tokens(), diagnostics.New(")"))

	expr := p.parseExpression()

	if expr != nil {
		t.Fatal("expected nil")
	}

	if len(p.reporter.Diagnostics()) == 0 {
		t.Fatal("expected diagnostic")
	}
}

func TestParseExpression_ListLiteral(t *testing.T) {
	expr := getExprValue(t, `["a", "b", 1]`)

	list, ok := expr.(*ListExpr)
	if !ok {
		t.Fatalf("got %T, want *ast.ListExpr", expr)
	}
	if len(list.Elements) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(list.Elements))
	}
	if s, ok := list.Elements[0].(*StringLiteral); !ok || s.Value != "a" {
		t.Errorf("element 0 = %+v, want StringLiteral(a)", list.Elements[0])
	}
	if i, ok := list.Elements[2].(*IntLiteral); !ok || i.Value != 1 {
		t.Errorf("element 2 = %+v, want IntLiteral(1)", list.Elements[2])
	}
}

func TestParseExpression_EmptyList(t *testing.T) {
	expr := getExprValue(t, `[]`)
	list, ok := expr.(*ListExpr)
	if !ok || len(list.Elements) != 0 {
		t.Fatalf("got %+v, want empty *ast.ListExpr", expr)
	}
}

func TestParseExpression_ListTrailingComma(t *testing.T) {
	expr := getExprValue(t, `["a", "b",]`)
	list, ok := expr.(*ListExpr)
	if !ok || len(list.Elements) != 2 {
		t.Fatalf("got %+v, want 2-element *ast.ListExpr", expr)
	}
}

func TestParseExpression_ObjectLiteral(t *testing.T) {
	expr := getExprValue(t, `{
    role_arn     = "arn:aws:iam::123:role/foo"
    session_name = "session"
  }`)

	obj, ok := expr.(*ObjectExpr)
	if !ok {
		t.Fatalf("got %T, want *ast.ObjectExpr", expr)
	}
	if len(obj.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(obj.Fields))
	}
	if obj.Fields[0].Name.Name != "role_arn" {
		t.Errorf("field 0 name = %q, want role_arn", obj.Fields[0].Name.Name)
	}
}

func TestParseExpression_NestedListsAndObjects(t *testing.T) {
	expr := getExprValue(t, `{
    tags = ["a", "b"]
  }`)
	obj := expr.(*ObjectExpr)
	list, ok := obj.Fields[0].Value.(*ListExpr)
	if !ok || len(list.Elements) != 2 {
		t.Fatalf("expected nested ListExpr with 2 elements, got %+v", obj.Fields[0].Value)
	}
}

func TestParseExpression_UnclosedListReportsError(t *testing.T) {
	full := "provider aws {\n  x = [\"a\", \"b\"\n}"
	_, reporter := parse(t, full)
	if !reporter.HasErrors() {
		t.Fatal("expected an error for unclosed list literal")
	}
}

// reference
func TestParsePostfix_SimpleMemberAccess(t *testing.T) {
	expr := getExprValue(t, "ubuntu.id")
	m, ok := expr.(*MemberExpr)
	if !ok {
		t.Fatalf("got %T, want *MemberExpr", expr)
	}
	base, ok := m.Object.(*Identifier)
	if !ok || base.Name != "ubuntu" {
		t.Errorf("Object = %+v, want Identifier(ubuntu)", m.Object)
	}
	if m.Property != "id" {
		t.Errorf("Property = %q, want id", m.Property)
	}
}

func TestParsePostfix_ChainedMemberAccess(t *testing.T) {
	// aws.east.region -> MemberExpr(MemberExpr(aws, east), region)
	expr := getExprValue(t, "aws.east.region")
	outer, ok := expr.(*MemberExpr)
	if !ok || outer.Property != "region" {
		t.Fatalf("outer = %+v, want Property=region", expr)
	}
	inner, ok := outer.Object.(*MemberExpr)
	if !ok || inner.Property != "east" {
		t.Fatalf("inner = %+v, want Property=region, ok=%v", outer.Object, ok)
	}
	base, ok := inner.Object.(*Identifier)
	if !ok || base.Name != "aws" {
		t.Errorf("base = %+v, want Identifier(aws)", inner.Object)
	}
}

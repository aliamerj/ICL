package parser

import (
	"testing"

	"github.com/aliamerj/icl/diagnostics"
	"github.com/aliamerj/icl/lexer"
)

func parseExpr(t *testing.T, input string) Expression {
	t.Helper()

	l := lexer.New(input)
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
	l := lexer.New(")")
	p := New(l.Tokens(), diagnostics.New(")"))

	expr := p.parseExpression()

	if expr != nil {
		t.Fatal("expected nil")
	}

	if len(p.reporter.Diagnostics()) == 0 {
		t.Fatal("expected diagnostic")
	}
}

package parser

import (
	"testing"

	"github.com/aliamerj/icl/diagnostics"
	"github.com/aliamerj/icl/lexer"
	"github.com/aliamerj/icl/tokens"
)

func parseProvider(t *testing.T, input string) *Block {
	t.Helper()

	l := lexer.New(input, diagnostics.New(input))
	p := New(l.Tokens(), diagnostics.New(input))

	block := p.parseBlock(tokens.PROVIDER)
	if block == nil {
		t.Fatal("expected provider block")
	}

	if len(p.reporter.Diagnostics()) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", p.reporter.Diagnostics())
	}

	return block
}

func TestParseProviderBlock_Empty(t *testing.T) {
	block := parseProvider(t, `
provider aws {
}
`)

	if block.Keyword != tokens.PROVIDER {
		t.Fatalf("expected provider, got %q", block.Keyword)
	}

	if len(block.Labels) != 1 {
		t.Fatalf("expected one label")
	}

	if block.Labels[0].Name != "aws" {
		t.Fatalf("expected aws")
	}

	if len(block.Body.Statements) != 0 {
		t.Fatalf("expected empty body")
	}
}

func TestParseProviderBlock_OneAttribute(t *testing.T) {
	block := parseProvider(t, `
provider aws {
  region = "us-east-1"
}
`)

	if len(block.Body.Statements) != 1 {
		t.Fatalf("expected one statement")
	}

	attr := block.Body.Statements[0].(*Attribute)

	if attr.Name.Name != "region" {
		t.Fatalf("expected region")
	}

	str, ok := attr.Value.(*StringLiteral)
	if !ok {
		t.Fatalf("expected string literal, got %T", attr.Value)
	}

	if str.Value != "us-east-1" {
		t.Fatalf("wrong value")
	}
}

func TestParseProviderBlock_MultipleAttributes(t *testing.T) {
	block := parseProvider(t, `
provider aws {
  region = "us-east-1"
  retries = 3
  enabled = true
}
`)

	if len(block.Body.Statements) != 3 {
		t.Fatalf("expected 3 attributes")
	}

	tests := []struct {
		name string
		typ  any
	}{
		{"region", &StringLiteral{}},
		{"retries", &IntLiteral{}},
		{"enabled", &BoolLiteral{}},
	}

	for i, tt := range tests {
		attr := block.Body.Statements[i].(*Attribute)

		if attr.Name.Name != tt.name {
			t.Fatalf("expected %s", tt.name)
		}

		switch tt.typ.(type) {
		case *StringLiteral:
			if _, ok := attr.Value.(*StringLiteral); !ok {
				t.Fatalf("expected string")
			}
		case *IntLiteral:
			if _, ok := attr.Value.(*IntLiteral); !ok {
				t.Fatalf("expected int")
			}
		case *BoolLiteral:
			if _, ok := attr.Value.(*BoolLiteral); !ok {
				t.Fatalf("expected bool")
			}
		}
	}
}

func TestParseProviderBlock_Expression(t *testing.T) {
	block := parseProvider(t, `
provider aws {
  value = 1 + 2 * 3
}
`)

	attr := block.Body.Statements[0].(*Attribute)

	root := attr.Value.(*BinaryExpr)

	if root.Operator != "+" {
		t.Fatalf("expected +")
	}

	if _, ok := root.Left.(*IntLiteral); !ok {
		t.Fatal("left should be int")
	}

	right := root.Right.(*BinaryExpr)

	if right.Operator != "*" {
		t.Fatalf("expected *")
	}
}

func TestParseProviderBlock_MissingLabel(t *testing.T) {
	input := `
provider {
}
`
	l := lexer.New(input, diagnostics.New(input))
	p := New(l.Tokens(), diagnostics.New(input))

	block := p.parseBlock(tokens.PROVIDER)

	if block != nil {
		t.Fatal("expected nil")
	}

	if len(p.reporter.Diagnostics()) == 0 {
		t.Fatal("expected diagnostics")
	}
}

func TestParseProviderBlock_MissingLeftBrace(t *testing.T) {
	input := `
provider aws
`
	l := lexer.New(input, diagnostics.New(input))
	p := New(l.Tokens(), diagnostics.New(input))
	block := p.parseBlock(tokens.PROVIDER)

	if block != nil {
		t.Fatal("expected nil")
	}

	if len(p.reporter.Diagnostics()) == 0 {
		t.Fatal("expected diagnostics")
	}
}

func TestParseProviderBlock_MissingEqual(t *testing.T) {
	input := `
provider aws {
    region "us-east-1"
}
`
	l := lexer.New(input, diagnostics.New(input))
	p := New(l.Tokens(), diagnostics.New(input))

	block := p.parseBlock(tokens.PROVIDER)

	if block != nil {
		t.Fatal("expected nil")
	}

	if len(p.reporter.Diagnostics()) == 0 {
		t.Fatal("expected diagnostics")
	}
}

func TestParseProviderBlock_MissingValue(t *testing.T) {
	input := `
provider aws {
    region =
}
`
	l := lexer.New(input, diagnostics.New(input))
	p := New(l.Tokens(), diagnostics.New(input))

	block := p.parseBlock(tokens.PROVIDER)

	if block != nil {
		t.Fatal("expected nil")
	}

	if len(p.reporter.Diagnostics()) == 0 {
		t.Fatal("expected diagnostics")
	}
}

func TestParseResource_HappyPath(t *testing.T) {
	src := `
resource aws_instance as app_server {
  ami           = "ami-123456"
  instance_type = "t2.micro"
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
		t.Fatalf("expected *ast.Block, got %T", prog.Statements[0])
	}
	if block.Keyword != tokens.RESOURCE {
		t.Errorf("Keyword = %q, want resource", block.Keyword)
	}
	if len(block.Labels) != 1 || block.Labels[0].Name != "aws_instance" {
		t.Fatalf("Labels = %+v, want [aws_instance]", block.Labels)
	}
	if block.Name == nil || block.Name.Name != "app_server" {
		t.Fatalf("Name = %+v, want app_server", block.Name)
	}
	if len(block.Body.Statements) != 2 {
		t.Fatalf("expected 2 attributes, got %d", len(block.Body.Statements))
	}

	attr0 := block.Body.Statements[0].(*Attribute)
	if attr0.Name.Name != "ami" {
		t.Errorf("attr0.Name = %q, want ami", attr0.Name.Name)
	}
}

func TestParseAttribute_KeywordUsableAsFieldName(t *testing.T) {
	src := `resource aws_instance as app_server {
  provider = aws.east
}`
	prog, reporter := parse(t, src)
	if reporter.HasErrors() {
		t.Fatalf("unexpected errors: %+v", reporter.Diagnostics())
	}
	block := prog.Statements[0].(*Block)
	attr := block.Body.Statements[0].(*Attribute)
	if attr.Name.Name != "provider" {
		t.Errorf("attr.Name = %q, want provider", attr.Name.Name)
	}
}

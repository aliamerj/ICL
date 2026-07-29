package eval

import (
	"testing"

	"github.com/aliamerj/icl/diagnostics"
	"github.com/aliamerj/icl/lexer"
	"github.com/aliamerj/icl/parser"
)

func TestEvalResource_HappyPath(t *testing.T) {
	src := `resource aws_instance as app_server {
  ami           = "ami-123456"
  instance_type = "t2.micro"
}`

	block := parseBlock(t, src)
	env := NewEnv()
	reporter := diagnostics.New(src)

	evalResource(block, env, reporter)
	cfg, _ := env.Registry.Resources.Lookup("app_server")
	if reporter.HasErrors() {
		t.Fatalf("unexpected errors: %+v", reporter.Diagnostics())
	}
	if cfg.Type != "aws_instance" || cfg.Name != "app_server" {
		t.Errorf("cfg = %+v", cfg)
	}
	if cfg.Extra["ami"].Str != "ami-123456" {
		t.Errorf("Extra[ami] = %+v", cfg.Extra["ami"])
	}
}

func TestEvalResource_DuplicateNameFails(t *testing.T) {
	env := NewEnv()
	reporter := diagnostics.New("")

	block1 := parseBlock(t, `resource aws_instance as app_server { ami = "a" }`)
	block2 := parseBlock(t, `resource aws_subnet as app_server { cidr_block = "10.0.0.0/24" }`)

	evalResource(block1, env, reporter)
	evalResource(block2, env, reporter)

	if !reporter.HasErrors() {
		t.Fatal("expected a duplicate-name error for the second block")
	}

	diags := reporter.Diagnostics()
	if len(diags) == 0 || diags[len(diags)-1].Code != diagnostics.DUPLICATE_NAME {
		t.Fatalf("expected duplicate-name diagnostic, got %+v", diags)
	}

	cfg2, _ := env.Registry.Resources.Lookup("app_server")
	if cfg2 == nil {
		t.Fatal("expected the first resource to remain registered")
	}
	if cfg2.Type != "aws_instance" {
		t.Errorf("expected the original resource to stay registered, got %+v", cfg2)
	}
}

func TestEvalResource_ProviderMetaAttributeResolves(t *testing.T) {
	env := NewEnv()

	env.Registry.Providers.Add(&ProviderConfig{
		Name:  "aws",
		Alias: "east",
	})

	reporter := diagnostics.New("")

	block := parseBlock(t, `resource aws_instance as app_server {
  ami      = "ami-123456"
  provider = aws.east
}`)

	evalResource(block, env, reporter)
	cfg, _ := env.Registry.Resources.Lookup("app_server")

	if reporter.HasErrors() {
		t.Fatalf("unexpected errors: %+v", reporter.Diagnostics())
	}
	if cfg.Provider != "aws.east" {
		t.Errorf("cfg.Provider = %q, want aws.east", cfg.Provider)
	}
	if _, exists := cfg.Extra["provider"]; exists {
		t.Error("provider should not leak into Extra")
	}
}

func TestEvalResource_UndefinedProviderReported(t *testing.T) {
	env := NewEnv()
	reporter := diagnostics.New("")

	block := parseBlock(t, `resource aws_instance as app_server {
  provider = aws.east
}`)
	evalResource(block, env, reporter)

	if !reporter.HasErrors() {
		t.Fatal("expected an undefined-provider error")
	}
}

func parseBlock(t *testing.T, src string) *parser.Block {
	t.Helper()
	r := diagnostics.New(src)
	l := lexer.New(src, r)
	p := parser.New(l.Tokens(), r)
	prog := p.ParseProgram()

	for _, stmt := range prog.Statements {
		block, ok := stmt.(*parser.Block)
		if !ok {
			continue
		}
		return block
	}
	return nil
}

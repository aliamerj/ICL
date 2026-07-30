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
	cfg, _ := env.Registry.Resources.lookup("app_server")
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

	cfg2, _ := env.Registry.Resources.lookup("app_server")
	if cfg2 == nil {
		t.Fatal("expected the first resource to remain registered")
	}
	if cfg2.Type != "aws_instance" {
		t.Errorf("expected the original resource to stay registered, got %+v", cfg2)
	}
}

func TestEvalResource_ProviderMetaAttributeResolves(t *testing.T) {
	env := NewEnv()

	env.Registry.Providers.add(&ProviderConfig{
		Name:  "aws",
		Alias: "east",
	})

	reporter := diagnostics.New("")

	block := parseBlock(t, `resource aws_instance as app_server {
  ami      = "ami-123456"
  provider = aws.east
}`)

	evalResource(block, env, reporter)
	cfg, _ := env.Registry.Resources.lookup("app_server")

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

func TestEvalResource_ReferencesAnotherResource(t *testing.T) {
	env := NewEnv()
	reporter := diagnostics.New("")

	vpcBlock := parseBlock(t, `resource aws_vpc as demo_vpc {
  cidr_block = "10.0.0.0/16"
}`)
	evalResource(vpcBlock, env, reporter)

	subnetBlock := parseBlock(t, `resource aws_subnet as public_subnet {
  vpc_id     = demo_vpc.id
  cidr_block = "10.0.0.0/24"
}`)
	evalResource(subnetBlock, env, reporter)

	subnetCfg, _ := env.Registry.Resources.lookup("public_subnet")

	if reporter.HasErrors() {
		t.Fatalf("unexpected errors: %+v", reporter.Diagnostics())
	}
	vpcID := subnetCfg.Extra["vpc_id"]
	if vpcID.Kind != KindRef || vpcID.Str != "aws_vpc.demo_vpc.id" {
		t.Errorf("vpc_id = %+v, want KindRef aws_vpc.demo_vpc.id", vpcID)
	}
}

func TestEvalResource_ForwardReferenceFails(t *testing.T) {
	// Mirrors the provider forward-reference limitation: a resource can
	// only reference resources declared BEFORE it, since the registry
	// is built incrementally in file order.
	env := NewEnv()
	reporter := diagnostics.New("")

	subnetBlock := parseBlock(t, `resource aws_subnet as public_subnet {
  vpc_id = demo_vpc.id
}`)
	evalResource(subnetBlock, env, reporter) // demo_vpc not registered yet

	if !reporter.HasErrors() {
		t.Fatal("expected a forward-reference error")
	}
}

func parseBlock(t *testing.T, src string) *parser.Block {
	t.Helper()
	r := diagnostics.New(src)
	scan := lexer.New(src, r)
	p := parser.New(scan.Tokens(), r)
	prog := p.ParseProgram()

	if r.HasErrors() {
		t.Fatalf("unexpected parse errors: %+v", r.Diagnostics())
	}
	if len(prog.Statements) != 1 {
		t.Fatalf("expected 1 top-level statement, got %d", len(prog.Statements))
	}
	block, ok := prog.Statements[0].(*parser.Block)
	if !ok {
		t.Fatalf("expected *ast.Block, got %T", prog.Statements[0])
	}
	return block
}

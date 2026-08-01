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

func TestEvalLookup_HappyPath(t *testing.T) {
	env := NewEnv()
	reporter := diagnostics.New("")

	block := parseBlock(t, `lookup aws_ami as ubuntu {
  most_recent = true
  owners      = ["099720109477"]
  filter = {
    name   = "name"
    values = ["ubuntu-*"]
  }
}`)
	evalLookup(block, env, reporter)

	if reporter.HasErrors() {
		t.Fatalf("unexpected errors: %+v", reporter.Diagnostics())
	}

	cfg, _ := env.Registry.Resources.lookup("ubuntu")
	if cfg.Kind != KindLookup {
		t.Errorf("Kind = %v, want KindLookup", cfg.Kind)
	}
	if cfg.Type != "aws_ami" || cfg.Name != "ubuntu" {
		t.Errorf("cfg = %+v", cfg)
	}
	if cfg.Extra["most_recent"].Bool != true {
		t.Errorf("Extra[most_recent] = %+v, want true", cfg.Extra["most_recent"])
	}
	owners := cfg.Extra["owners"]
	if owners.Kind != KindList || owners.List[0].Str != "099720109477" {
		t.Errorf("Extra[owners] = %+v", owners)
	}
	filter := cfg.Extra["filter"]
	if filter.Kind != KindObject || filter.Object["name"].Str != "name" {
		t.Errorf("Extra[filter] = %+v", filter)
	}
}

func TestEvalLookup_MissingNameOrLabelReportsError(t *testing.T) {
	// Guards evalDeclaration's own invariant checks, independent of the
	// parser already enforcing them — same defensive posture as EvalResource.
	env := NewEnv()
	reporter := diagnostics.New("")

	badBlock := parseBlock(t, `lookup aws_ami as ubuntu {}`)
	badBlock.Name = nil // simulate the invariant being violated directly

	evalLookup(badBlock, env, reporter)

	cfg, _ := env.Registry.Resources.lookup("ubuntu")
	if !reporter.HasErrors() || cfg != nil {
		t.Fatal("expected an error and nil config when Name is missing")
	}
}

func TestEvalLookup_ProviderMetaAttributeResolves(t *testing.T) {
	env := NewEnv()
	env.Registry.Providers.add(&ProviderConfig{Name: "aws", Alias: "east"})
	reporter := diagnostics.New("")

	block := parseBlock(t, `lookup aws_ami as ubuntu {
  most_recent = true
  provider    = aws.east
}`)
	evalLookup(block, env, reporter)

	if reporter.HasErrors() {
		t.Fatalf("unexpected errors: %+v", reporter.Diagnostics())
	}
	cfg, _ := env.Registry.Resources.lookup("ubuntu")
	if cfg.Provider != "aws.east" {
		t.Errorf("cfg.Provider = %q, want aws.east", cfg.Provider)
	}
	if _, exists := cfg.Extra["provider"]; exists {
		t.Error("provider should not leak into Extra")
	}
}

// --- Flat namespace: resource and lookup share ONE registry ---

func TestEvalLookup_NameCollidesWithResource(t *testing.T) {
	env := NewEnv()
	reporter := diagnostics.New("")

	resourceBlock := parseBlock(t, `resource aws_instance as ubuntu {
  ami = "ami-123"
}`)
	lookupBlock := parseBlock(t, `lookup aws_ami as ubuntu {
  most_recent = true
}`)

	evalResource(resourceBlock, env, reporter)
	evalLookup(lookupBlock, env, reporter)

	if !reporter.HasErrors() {
		t.Fatal("expected a duplicate-name error: 'ubuntu' already used by a resource")
	}

	cfg, _ := env.Registry.Resources.lookup("ubuntu")
	if cfg == nil {
		t.Fatal("expected the original resource to remain registered")
	}
	if cfg.Kind != KindResource || cfg.Type != "aws_instance" {
		t.Fatalf("expected the resource declaration to win, got %+v", cfg)
	}
}

func TestEvalLookup_NameCollidesWithAnotherLookup(t *testing.T) {
	env := NewEnv()
	reporter := diagnostics.New("")

	b1 := parseBlock(t, `lookup aws_ami as ubuntu { most_recent = true }`)
	b2 := parseBlock(t, `lookup aws_ami as ubuntu { most_recent = false }`)

	evalLookup(b1, env, reporter)
	evalLookup(b2, env, reporter)

	cfg, _ := env.Registry.Resources.lookup("ubuntu")
	if !reporter.HasErrors() {
		t.Fatal("expected a duplicate-name error for the second lookup")
	}
	if cfg == nil {
		t.Fatal("expected the first lookup to remain registered")
	}
	if cfg.Kind != KindLookup || cfg.Extra["most_recent"].Bool != true {
		t.Fatalf("expected the first lookup to remain registered, got %+v", cfg)
	}
}

// --- Reference resolution: the actual point of this session ---

func TestEval_LookupReferenceUsesDataPrefix(t *testing.T) {
env := NewEnv()
	env.Registry.Resources.add(&ResourceConfig{Kind: KindLookup, Type: "aws_ami", Name: "ubuntu"})

	v, ok := eval(memberExpr("ubuntu", "id"), env, diagnostics.New(""))
	if !ok {
		t.Fatal("expected success")
	}
	if v.Kind != KindRef {
		t.Fatalf("Kind = %v, want KindRef", v.Kind)
	}
	if v.Str != "data.aws_ami.ubuntu.id" {
		t.Errorf("Str = %q, want data.aws_ami.ubuntu.id", v.Str)
	}
}

func TestEval_ResourceReferenceStillHasNoDataPrefix(t *testing.T) {
	// Regression: confirms the Kind branch didn't accidentally apply
	// "data." to plain resources too.
		env := NewEnv()
	env.Registry.Resources.add(&ResourceConfig{Kind: KindResource, Type: "aws_vpc", Name: "demo_vpc"})

	v, ok := eval(memberExpr("demo_vpc", "id"), env, diagnostics.New(""))
	if !ok || v.Str != "aws_vpc.demo_vpc.id" {
		t.Fatalf("got %+v, ok=%v, want aws_vpc.demo_vpc.id (no data. prefix)", v, ok)
	}
}

func TestEvalResource_ReferencesLookupResult(t *testing.T) {
	// Integration: the real AMI-lookup-into-instance case, end to end
	// through the actual parser, exactly like the file that passed
	// tofu validate.
	env := NewEnv()
	reporter := diagnostics.New("")

	lookupBlock := parseBlock(t, `lookup aws_ami as ubuntu {
  most_recent = true
}`)
	evalLookup(lookupBlock, env, reporter)

	resourceBlock := parseBlock(t, `resource aws_instance as app_server {
  ami           = ubuntu.id
  instance_type = "t2.micro"
}`)
	evalResource(resourceBlock, env, reporter)

	if reporter.HasErrors() {
		t.Fatalf("unexpected errors: %+v", reporter.Diagnostics())
	}
	
  resCfg, _ := env.Registry.Resources.lookup("app_server")

	ami := resCfg.Extra["ami"]
	if ami.Kind != KindRef || ami.Str != "data.aws_ami.ubuntu.id" {
		t.Errorf("ami = %+v, want KindRef data.aws_ami.ubuntu.id", ami)
	}
}

func TestEvalLookup_ForwardReferenceFails(t *testing.T) {
	env := NewEnv()
	reporter := diagnostics.New("")

	resourceBlock := parseBlock(t, `resource aws_instance as app_server {
  ami = ubuntu.id
}`)
	evalResource(resourceBlock, env, reporter) // 'ubuntu' lookup not declared yet

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

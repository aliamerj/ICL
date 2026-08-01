package parser

import (
	"testing"

	"github.com/aliamerj/icl/tokens"
)

// --- Parser: var declaration, all forms ---

func TestParseVar_BareNoTypeNoDefault(t *testing.T) {
	src := `var instance_name {
}`
	prog, reporter := parse(t, src)
	if reporter.HasErrors() {
		t.Fatalf("unexpected errors: %+v", reporter.Diagnostics())
	}
	decl, ok := prog.Statements[0].(*VarDecl)
	if !ok {
		t.Fatalf("expected *ast.VarDecl, got %T", prog.Statements[0])
	}
	if decl.Name.Name != "instance_name" {
		t.Errorf("Name = %q, want instance_name", decl.Name.Name)
	}
	if decl.Type != nil {
		t.Errorf("Type = %+v, want nil (no `is` given)", decl.Type)
	}
	if decl.Default != nil {
		t.Errorf("Default = %+v, want nil", decl.Default)
	}
}

func TestParseVar_BareIdentifierNoBraces(t *testing.T) {
	src := `var instance_name`
	prog, reporter := parse(t, src)
	if reporter.HasErrors() {
		t.Fatalf("unexpected errors: %+v", reporter.Diagnostics())
	}
	decl := prog.Statements[0].(*VarDecl)
	if decl.Name.Name != "instance_name" || decl.Type != nil || decl.Default != nil || decl.Body != nil {
		t.Errorf("decl = %+v, want fully bare", decl)
	}
}

func TestParseVar_WithTypeNoDefault(t *testing.T) {
	src := `var instance_type is string`
	prog, reporter := parse(t, src)
	if reporter.HasErrors() {
		t.Fatalf("unexpected errors: %+v", reporter.Diagnostics())
	}
	decl := prog.Statements[0].(*VarDecl)
	if decl.Type == nil || decl.Type.Name != "string" {
		t.Fatalf("Type = %+v, want string", decl.Type)
	}
	if decl.Default != nil {
		t.Error("expected no default")
	}
}

func TestParseVar_DefaultWithoutIs(t *testing.T) {
	src := `var bucket_name = "my_default_bucket_name"`
	prog, reporter := parse(t, src)
	if reporter.HasErrors() {
		t.Fatalf("unexpected errors: %+v", reporter.Diagnostics())
	}
	decl := prog.Statements[0].(*VarDecl)
	if decl.Type != nil {
		t.Errorf("Type = %+v, want nil (is omitted)", decl.Type)
	}
	str, ok := decl.Default.(*StringLiteral)
	if !ok || str.Value != "my_default_bucket_name" {
		t.Fatalf("Default = %+v, want StringLiteral", decl.Default)
	}
}

func TestParseVar_TypeAndDefault(t *testing.T) {
	src := `var instance_type is string = "t2.micro"`
	prog, reporter := parse(t, src)
	if reporter.HasErrors() {
		t.Fatalf("unexpected errors: %+v", reporter.Diagnostics())
	}
	decl := prog.Statements[0].(*VarDecl)
	if decl.Type == nil || decl.Type.Name != "string" {
		t.Fatalf("Type = %+v, want string", decl.Type)
	}
	str, ok := decl.Default.(*StringLiteral)
	if !ok || str.Value != "t2.micro" {
		t.Fatalf("Default = %+v, want StringLiteral(t2.micro)", decl.Default)
	}
}

func TestParseVar_ConflictingTypeAndDefaultStillParsesCleanly(t *testing.T) {
	// Parser doesn't know or care that types conflict — it just captures
	// both, unevaluated. The mismatch is EvalVar's job to catch, not the
	// parser's. This test locks down that division of responsibility.
	src := `var port is string = 123`
	prog, reporter := parse(t, src)
	if reporter.HasErrors() {
		t.Fatalf("parser should not error on this — got: %+v", reporter.Diagnostics())
	}
	decl := prog.Statements[0].(*VarDecl)
	if decl.Type.Name != "string" {
		t.Errorf("Type = %q, want string", decl.Type.Name)
	}
	if _, ok := decl.Default.(*IntLiteral); !ok {
		t.Errorf("Default = %T, want *ast.IntLiteral(123)", decl.Default)
	}
}

func TestParseVar_BodyForm(t *testing.T) {
	src := `var bucket_name is string {
  description = "My variable used to set bucket name"
  default     = "my_default_bucket_name"
}`
	prog, reporter := parse(t, src)
	if reporter.HasErrors() {
		t.Fatalf("unexpected errors: %+v", reporter.Diagnostics())
	}
	decl := prog.Statements[0].(*VarDecl)
	if decl.Body == nil {
		t.Fatal("expected Body to be set")
	}
	if len(decl.Body.Statements) != 2 {
		t.Fatalf("expected 2 attributes in body, got %d", len(decl.Body.Statements))
	}
	if decl.Default != nil {
		t.Error("Default (shorthand) should be nil when body form is used")
	}
}

func TestParseVar_UnknownTypeStillParses(t *testing.T) {
	// Same principle as the conflict case: bareword after `is` is just an
	// identifier to the parser. Validity of the type name is EvalVar's job.
	src := `var x is not_a_real_type`
	prog, reporter := parse(t, src)
	if reporter.HasErrors() {
		t.Fatalf("unexpected parse errors: %+v", reporter.Diagnostics())
	}
	decl := prog.Statements[0].(*VarDecl)
	if decl.Type.Name != "not_a_real_type" {
		t.Errorf("Type = %q", decl.Type.Name)
	}
}

func TestParseVar_MissingNameIsAnError(t *testing.T) {
	src := `var is string`
	_, reporter := parse(t, src)
	if !reporter.HasErrors() {
		t.Fatal("expected an error: 'is' was consumed as the name, 'string' as the type keyword — should fail expecting IDENTIFIER for type or fail structurally")
	}
}

func TestParseVar_UnclosedBody(t *testing.T) {
	src := `var bucket_name is string {
  default = "x"
`
	_, reporter := parse(t, src)
	if !reporter.HasErrors() {
		t.Fatal("expected an error for unclosed body")
	}
}

func TestParseProgram_RecoversAcrossVarAndResource(t *testing.T) {
	src := `
var is string

resource aws_instance as app_server {
  ami = "ami-123"
}
`
	prog, reporter := parse(t, src)
	if !reporter.HasErrors() {
		t.Fatal("expected an error from the broken var declaration")
	}
	if len(prog.Statements) != 1 {
		t.Fatalf("expected 1 recovered statement, got %d", len(prog.Statements))
	}
	block, ok := prog.Statements[0].(*Block)
	if !ok || block.Keyword != tokens.RESOURCE {
		t.Errorf("recovered statement wrong: %+v", prog.Statements[0])
	}
}

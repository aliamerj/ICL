package parser

import (
	"testing"

	"github.com/aliamerj/icl/tokens"
)

// --- Parser: output declaration ---

func TestParseOutput_ShorthandForm(t *testing.T) {
	src := `output bucket_id = my_bucket.id`
	prog, reporter := parse(t, src)
	if reporter.HasErrors() {
		t.Fatalf("unexpected errors: %+v", reporter.Diagnostics())
	}
	decl, ok := prog.Statements[0].(*OutputDecl)
	if !ok {
		t.Fatalf("expected *ast.OutputDecl, got %T", prog.Statements[0])
	}
	if decl.Name.Name != "bucket_id" {
		t.Errorf("Name = %q, want bucket_id", decl.Name.Name)
	}
	member, ok := decl.Value.(*MemberExpr)
	if !ok {
		t.Fatalf("Value = %T, want *ast.MemberExpr", decl.Value)
	}
	base, ok := member.Object.(*Identifier)
	if !ok || base.Name != "my_bucket" || member.Property != "id" {
		t.Errorf("Value = %+v, want my_bucket.id", member)
	}
	if decl.Body != nil {
		t.Error("Body should be nil for shorthand form")
	}
}

func TestParseOutput_ShorthandWithLiteral(t *testing.T) {
	src := `output greeting = "hello world"`
	prog, reporter := parse(t, src)
	if reporter.HasErrors() {
		t.Fatalf("unexpected errors: %+v", reporter.Diagnostics())
	}
	decl := prog.Statements[0].(*OutputDecl)
	str, ok := decl.Value.(*StringLiteral)
	if !ok || str.Value != "hello world" {
		t.Fatalf("Value = %+v, want StringLiteral(hello world)", decl.Value)
	}
}

func TestParseOutput_BodyForm(t *testing.T) {
	src := `output bucket_arn {
  value       = my_bucket.id
  description = "The bucket's identifier"
  sensitive   = false
}`
	prog, reporter := parse(t, src)
	if reporter.HasErrors() {
		t.Fatalf("unexpected errors: %+v", reporter.Diagnostics())
	}
	decl := prog.Statements[0].(*OutputDecl)
	if decl.Body == nil {
		t.Fatal("expected Body to be set")
	}
	if len(decl.Body.Statements) != 3 {
		t.Fatalf("expected 3 attributes, got %d", len(decl.Body.Statements))
	}
	if decl.Value != nil {
		t.Error("Value (shorthand) should be nil when body form is used")
	}
}

func TestParseOutput_MissingValueIsAnError(t *testing.T) {
	// Neither `=` nor `{` follows the name — genuinely invalid, unlike
	// var, which allows a fully bare form.
	src := `output bucket_id`
	prog, reporter := parse(t, src)
	if !reporter.HasErrors() {
		t.Fatal("expected an error: output must have a value")
	}
	if len(prog.Statements) != 0 {
		t.Errorf("expected 0 statements after failed decl, got %d", len(prog.Statements))
	}
}

func TestParseOutput_MissingName(t *testing.T) {
	src := `output = "x"`
	_, reporter := parse(t, src)
	if !reporter.HasErrors() {
		t.Fatal("expected an error for a missing output name")
	}
}

func TestParseOutput_UnclosedBody(t *testing.T) {
	src := `output bucket_arn {
  value = my_bucket.id
`
	_, reporter := parse(t, src)
	if !reporter.HasErrors() {
		t.Fatal("expected an error for an unclosed body")
	}
}

func TestParseOutput_Range(t *testing.T) {
	src := `output bucket_id = my_bucket.id`
	prog, reporter := parse(t, src)
	if reporter.HasErrors() {
		t.Fatalf("unexpected errors: %+v", reporter.Diagnostics())
	}
	decl := prog.Statements[0].(*OutputDecl)
	if decl.Rng.Start.Offset != 0 {
		t.Errorf("Rng.Start.Offset = %d, want 0", decl.Rng.Start.Offset)
	}
	if decl.Rng.End.Offset <= decl.Rng.Start.Offset {
		t.Errorf("Rng.End (%d) should be after Rng.Start (%d)", decl.Rng.End.Offset, decl.Rng.Start.Offset)
	}
}

func TestParseProgram_RecoversAcrossOutputAndResource(t *testing.T) {
	src := `
output bucket_id

resource aws_instance as app_server {
  ami = "ami-123"
}
`
	prog, reporter := parse(t, src)
	if !reporter.HasErrors() {
		t.Fatal("expected an error from the broken output declaration")
	}
	if len(prog.Statements) != 1 {
		t.Fatalf("expected 1 recovered statement, got %d", len(prog.Statements))
	}
	block, ok := prog.Statements[0].(*Block)
	if !ok || block.Keyword != tokens.RESOURCE {
		t.Errorf("recovered statement wrong: %+v", prog.Statements[0])
	}
}

package parser

import (
	"testing"

	"github.com/aliamerj/icl/diagnostics"
	"github.com/aliamerj/icl/lexer"
)

func parse(t *testing.T, source string) (*Program, *diagnostics.Reporter) {
	t.Helper()
	scan := lexer.New(source)
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
	if block.Keyword != "provider" {
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
		t.Fatalf("expected recovery to still parse 1 valid statement, got %d", len(prog.Statements))
	}
	block := prog.Statements[0].(*Block)
	if block.Keyword != "provider" || len(block.Labels) != 1 || block.Labels[0].Name != "aws" {
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

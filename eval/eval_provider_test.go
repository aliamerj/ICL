package eval

import (
	"testing"

	"github.com/aliamerj/icl/diagnostics"
	"github.com/aliamerj/icl/lexer"
	"github.com/aliamerj/icl/parser"
)

// parseProviderBlock is a test helper: runs the real lexer + parser
// and returns the first top-level *ast.Block, so eval tests exercise
// the actual pipeline instead of hand-building AST nodes.
func parseProviderBlock(t *testing.T, source string) (*parser.Block, *diagnostics.Reporter) {
	t.Helper()

	scan := lexer.New(source, diagnostics.New(source))
	parseReporter := diagnostics.New(source)
	p := parser.New(scan.Tokens(), parseReporter)
	prog := p.ParseProgram()

	if parseReporter.HasErrors() {
		t.Fatalf("unexpected parse errors: %+v", parseReporter.Diagnostics())
	}
	if len(prog.Statements) != 1 {
		t.Fatalf("expected 1 top-level statement, got %d", len(prog.Statements))
	}

	block, ok := prog.Statements[0].(*parser.Block)
	if !ok {
		t.Fatalf("expected *ast.Block, got %T", prog.Statements[0])
	}
	return block, parseReporter
}

func TestEvalProvider_HappyPath(t *testing.T) {
	src := `provider aws {
  source  = "hashicorp/aws"
  version = "5.37.0"
}`
	block, _ := parseProviderBlock(t, src)

	evalReporter := diagnostics.New(src)
	cfg := evalProvider(block, newEnv(), evalReporter)

	if evalReporter.HasErrors() {
		t.Fatalf("unexpected eval errors: %+v", evalReporter.Diagnostics())
	}
	if cfg == nil {
		t.Fatal("expected a non-nil ProviderConfig")
	}
	if cfg.Name != "aws" {
		t.Errorf("cfg.Name = %q, want aws", cfg.Name)
	}
	if cfg.Source != "hashicorp/aws" {
		t.Errorf("cfg.Source = %q, want hashicorp/aws", cfg.Source)
	}
	if cfg.Version != "5.37.0" {
		t.Errorf("cfg.Version = %q, want 5.37.0", cfg.Version)
	}
	if len(cfg.Extra) != 0 {
		t.Errorf("expected no extra fields, got %+v", cfg.Extra)
	}
}

func TestEvalProvider_MissingSource(t *testing.T) {
	src := `provider aws {
  version = "5.37.0"
}`
	block, _ := parseProviderBlock(t, src)

	evalReporter := diagnostics.New(src)
	cfg := evalProvider(block, newEnv(), evalReporter)

	if !evalReporter.HasErrors() {
		t.Fatal("expected an error for missing required field 'source'")
	}
	// cfg is still returned (not nil) so callers can inspect what DID resolve,
	// even though it's not safe to use for anything real — confirm that contract.
	if cfg == nil {
		t.Fatal("expected a non-nil ProviderConfig even when validation fails")
	}
	if cfg.Version != "5.37.0" {
		t.Errorf("cfg.Version = %q, want 5.37.0 (should still resolve despite the other error)", cfg.Version)
	}
}

func TestEvalProvider_SourceWrongType(t *testing.T) {
	src := `provider aws {
  source  = 5
  version = "5.37.0"
}`
	block, _ := parseProviderBlock(t, src)

	evalReporter := diagnostics.New(src)
	cfg := evalProvider(block, newEnv(), evalReporter)

	if !evalReporter.HasErrors() {
		t.Fatal("expected a type-mismatch error for source = 5")
	}
	if cfg.Source != "" {
		t.Errorf("cfg.Source = %q, want empty since the value was rejected", cfg.Source)
	}
	// the valid sibling field should still resolve — one bad attribute
	// shouldn't poison the rest of the block.
	if cfg.Version != "5.37.0" {
		t.Errorf("cfg.Version = %q, want 5.37.0", cfg.Version)
	}
}

func TestEvalProvider_VersionWrongType(t *testing.T) {
	src := `provider aws {
  source  = "hashicorp/aws"
  version = 5.0
}`
	block, _ := parseProviderBlock(t, src)

	evalReporter := diagnostics.New(src)
	cfg := evalProvider(block, newEnv(), evalReporter)

	if !evalReporter.HasErrors() {
		t.Fatal("expected a type-mismatch error for version = 5.0")
	}
	if cfg.Version != "" {
		t.Errorf("cfg.Version = %q, want empty since the value was rejected", cfg.Version)
	}
}

func TestEvalProvider_ExtraFieldsCaptured(t *testing.T) {
	src := `provider aws {
  source     = "hashicorp/aws"
  version    = "5.37.0"
  region     = "eu-west-1"
  maxRetries = 3
}`
	block, _ := parseProviderBlock(t, src)

	evalReporter := diagnostics.New(src)
	cfg := evalProvider(block, newEnv(), evalReporter)

	if evalReporter.HasErrors() {
		t.Fatalf("unexpected eval errors: %+v", evalReporter.Diagnostics())
	}

	region, ok := cfg.Extra["region"]
	if !ok {
		t.Fatal("expected 'region' to be captured in Extra")
	}
	if region.Kind != KindString || region.Str != "eu-west-1" {
		t.Errorf("Extra[region] = %+v, want KindString eu-west-1", region)
	}

	retries, ok := cfg.Extra["maxRetries"]
	if !ok {
		t.Fatal("expected 'maxRetries' to be captured in Extra")
	}
	if retries.Kind != KindInt || retries.Int != 3 {
		t.Errorf("Extra[maxRetries] = %+v, want KindInt 3", retries)
	}
}

func TestEvalProvider_MultipleErrorsAllReported(t *testing.T) {
	// Confirms eval doesn't stop at the first problem — same
	// accumulate-don't-abort philosophy as the parser.
	src := `provider aws {
  source  = 5
  version = 5.0
}`
	block, _ := parseProviderBlock(t, src)

	evalReporter := diagnostics.New(src)
	evalProvider(block, newEnv(), evalReporter)

	diags := evalReporter.Diagnostics()
	if len(diags) < 2 {
		t.Fatalf("expected at least 2 diagnostics (source and version type mismatches), got %d: %+v",
			len(diags), diags)
	}
}

func TestEvalProvider_EmptyBody(t *testing.T) {
	src := `provider aws {}`
	block, _ := parseProviderBlock(t, src)

	evalReporter := diagnostics.New(src)
	cfg := evalProvider(block, newEnv(), evalReporter)

	if !evalReporter.HasErrors() {
		t.Fatal("expected a missing-required-field error for an empty provider block")
	}
	if cfg.Name != "aws" {
		t.Errorf("cfg.Name = %q, want aws (label should still resolve)", cfg.Name)
	}
}

func TestEvalProvider_ArithmeticValueWrongType(t *testing.T) {
	src := `provider aws {
  source  = 2 + 3
  version = "1.0"
}`
	block, _ := parseProviderBlock(t, src)
	evalReporter := diagnostics.New(src)
	cfg := evalProvider(block, newEnv(), evalReporter)

	if !evalReporter.HasErrors() {
		t.Fatal("expected a type-mismatch error: source evaluated to an int, not a string")
	}
	if cfg.Source != "" {
		t.Errorf("cfg.Source = %q, want empty", cfg.Source)
	}
}

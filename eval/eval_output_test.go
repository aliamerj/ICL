package eval

import (
	"testing"

	"github.com/aliamerj/icl/diagnostics"
	"github.com/aliamerj/icl/lexer"
	"github.com/aliamerj/icl/parser"
)

// --- EvalOutput: shorthand form ---
func TestEvalOutput_ShorthandWithLiteral(t *testing.T) {
	env := NewEnv()
	reporter := diagnostics.New("")

	decl := parseSingleOutputDecl(t, `output greeting = "hello"`)
	evalOutput(decl, env, reporter)
	cfg, _ := env.Registry.Outputs.lookup("greeting")

	if reporter.HasErrors() {
		t.Fatalf("unexpected errors: %+v", reporter.Diagnostics())
	}
	if cfg.Name != "greeting" || cfg.Value.Str != "hello" {
		t.Errorf("cfg = %+v", cfg)
	}
}

func TestEvalOutput_ShorthandReferencesResourceStaysDeferred(t *testing.T) {
	env := NewEnv()
	reporter := diagnostics.New("")

	resourceBlock := parseBlock(t, `resource aws_s3_bucket as my_bucket {
  bucket = "my-app-bucket"
}`)
	evalResource(resourceBlock, env, reporter)

	decl := parseSingleOutputDecl(t, `output bucket_id = my_bucket.id`)
	evalOutput(decl, env, reporter)
	cfg, _ := env.Registry.Outputs.lookup("bucket_id")

	if reporter.HasErrors() {
		t.Fatalf("unexpected errors: %+v", reporter.Diagnostics())
	}
	if cfg.Value.Kind != KindRef || cfg.Value.Str != "aws_s3_bucket.my_bucket.id" {
		t.Errorf("Value = %+v, want KindRef aws_s3_bucket.my_bucket.id", cfg.Value)
	}
}

// --- EvalOutput: body form ---

func TestEvalOutput_BodyFormAllFields(t *testing.T) {
	env := NewEnv()
	reporter := diagnostics.New("")

	decl := parseSingleOutputDecl(t, `output bucket_arn {
  value       = "arn:aws:s3:::my-bucket"
  description = "The bucket's identifier"
  sensitive   = true
}`)
	evalOutput(decl, env, reporter)
	cfg, _ := env.Registry.Outputs.lookup("bucket_arn")

	if reporter.HasErrors() {
		t.Fatalf("unexpected errors: %+v", reporter.Diagnostics())
	}
	if cfg.Value.Str != "arn:aws:s3:::my-bucket" {
		t.Errorf("Value = %+v", cfg.Value)
	}
	if cfg.Description != "The bucket's identifier" {
		t.Errorf("Description = %q", cfg.Description)
	}
	if !cfg.Sensitive {
		t.Error("Sensitive = false, want true")
	}
}

func TestEvalOutput_BodyFormMissingValueIsError(t *testing.T) {
	env := NewEnv()
	reporter := diagnostics.New("")

	decl := parseSingleOutputDecl(t, `output bucket_arn {
  description = "no value set"
}`)
	evalOutput(decl, env, reporter)
	cfg, _ := env.Registry.Outputs.lookup("bucket_arn")

	if !reporter.HasErrors() || cfg != nil {
		t.Fatal("expected an error: body form output must set 'value'")
	}
}

func TestEvalOutput_DescriptionMustBeString(t *testing.T) {
	env := NewEnv()
	reporter := diagnostics.New("")

	decl := parseSingleOutputDecl(t, `output x {
  value       = "y"
  description = 123
}`)
	evalOutput(decl, env, reporter)

	if !reporter.HasErrors() {
		t.Fatal("expected a type-mismatch error for non-string description")
	}
}

func TestEvalOutput_SensitiveMustBeBool(t *testing.T) {
	env := NewEnv()
	reporter := diagnostics.New("")

	decl := parseSingleOutputDecl(t, `output x {
  value     = "y"
  sensitive = "yes"
}`)
	evalOutput(decl, env, reporter)

	if !reporter.HasErrors() {
		t.Fatal("expected a type-mismatch error for non-bool sensitive")
	}
}

func TestEvalOutput_UnknownAttributeReportsError(t *testing.T) {
	env := NewEnv()
	reporter := diagnostics.New("")

	decl := parseSingleOutputDecl(t, `output x {
  value    = "y"
  nonsense = "z"
}`)
	evalOutput(decl, env, reporter)

	if !reporter.HasErrors() {
		t.Fatal("expected an error for an unknown output attribute")
	}
}

// --- Duplicate detection ---

func TestEvalOutput_DuplicateNameFails(t *testing.T) {
	env := NewEnv()
	reporter := diagnostics.New("")

	d1 := parseSingleOutputDecl(t, `output bucket_id = "a"`)
	d2 := parseSingleOutputDecl(t, `output bucket_id = "b"`)

	evalOutput(d1, env, reporter)
	evalOutput(d2, env, reporter)

	if !reporter.HasErrors() {
		t.Fatal("expected a duplicate-name error for the second output")
	}
}

func TestEvalOutput_NamesAreASeparateNamespaceFromResources(t *testing.T) {
	// Deliberate design check: unlike resource/lookup/var (which share one
	// flat namespace), output names don't need to be globally unique against
	// resources — an output can share a name with the resource it exposes,
	// since Terraform's own `output` and `resource` blocks are namespaced
	// separately. Confirm this is the actual intended behavior.
	env := NewEnv()
	reporter := diagnostics.New("")

	resourceBlock := parseBlock(t, `resource aws_s3_bucket as my_bucket {
  bucket = "x"
}`)
	evalResource(resourceBlock, env, reporter)

	decl := parseSingleOutputDecl(t, `output my_bucket = my_bucket.id`)
	evalOutput(decl, env, reporter)

	if reporter.HasErrors() {
		t.Fatalf("unexpected errors — output/resource namespaces should be independent: %+v", reporter.Diagnostics())
	}
	cfg, _ := env.Registry.Outputs.lookup("my_bucket")

	if cfg == nil {
		t.Fatal("expected output to be declared successfully")
	}
}

// --- Value.Native() / tfjson serialization of KindRef via output ---

func TestOutputConfig_RefValueSerializesWithDollarBraces(t *testing.T) {
	env := NewEnv()
	reporter := diagnostics.New("")

	resourceBlock := parseBlock(t, `resource aws_s3_bucket as my_bucket {
  bucket = "x"
}`)
	evalResource(resourceBlock, env, reporter)

	decl := parseSingleOutputDecl(t, `output bucket_id = my_bucket.id`)
	evalOutput(decl, env, reporter)
	cfg, _ := env.Registry.Outputs.lookup("bucket_id")

	native := cfg.Value.Native()
	if native != "${aws_s3_bucket.my_bucket.id}" {
		t.Errorf("Native() = %v, want ${aws_s3_bucket.my_bucket.id}", native)
	}
}

func parseSingleOutputDecl(t *testing.T, source string) *parser.OutputDecl {
	t.Helper()
	parseReporter := diagnostics.New(source)
	scan := lexer.New(source, parseReporter)
	p := parser.New(scan.Tokens(), parseReporter)
	prog := p.ParseProgram()
	if parseReporter.HasErrors() {
		t.Fatalf("unexpected parse errors: %+v", parseReporter.Diagnostics())
	}
	decl, ok := prog.Statements[0].(*parser.OutputDecl)
	if !ok {
		t.Fatalf("expected *ast.OutputDecl, got %T", prog.Statements[0])
	}
	return decl
}

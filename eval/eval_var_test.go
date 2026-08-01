package eval

import (
	"testing"

	"github.com/aliamerj/icl/diagnostics"
	"github.com/aliamerj/icl/lexer"
	"github.com/aliamerj/icl/parser"
)

func parseSingleVarDecl(t *testing.T, source string) *parser.VarDecl {
	t.Helper()
	parseReporter := diagnostics.New(source)
	scan := lexer.New(source, parseReporter)
	p := parser.New(scan.Tokens(), parseReporter)
	prog := p.ParseProgram()
	if parseReporter.HasErrors() {
		t.Fatalf("unexpected parse errors: %+v", parseReporter.Diagnostics())
	}
	decl, ok := prog.Statements[0].(*parser.VarDecl)
	if !ok {
		t.Fatalf("expected *ast.VarDecl, got %T", prog.Statements[0])
	}
	return decl
}

// --- EvalVar: type inference and validation ---

func TestEvalVar_BareNoDefaultInfersAny(t *testing.T) {
	env := NewEnv()
	reporter := diagnostics.New("")

	decl := parseSingleVarDecl(t, `var instance_name`)
	evalVar(decl, env, reporter)
	cfg, ok := env.Registry.Vars.lookup("instance_name")

	if reporter.HasErrors() || !ok {
		t.Fatalf("unexpected errors: %+v", reporter.Diagnostics())
	}
	if cfg.Type != "any" {
		t.Errorf("Type = %q, want any", cfg.Type)
	}
	if cfg.HasDefault {
		t.Error("expected HasDefault = false")
	}
}

func TestEvalVar_DefaultWithoutIsInfersType(t *testing.T) {
	env := NewEnv()
	reporter := diagnostics.New("")

	decl := parseSingleVarDecl(t, `var bucket_name = "my_default_bucket_name"`)
	evalVar(decl, env, reporter)
	cfg, ok := env.Registry.Vars.lookup("bucket_name")

	if reporter.HasErrors() || !ok {
		t.Fatalf("unexpected errors: %+v", reporter.Diagnostics())
	}
	if cfg.Type != "string" {
		t.Errorf("Type = %q, want string (inferred)", cfg.Type)
	}
	if !cfg.HasDefault || cfg.Default.Str != "my_default_bucket_name" {
		t.Errorf("Default = %+v", cfg.Default)
	}
}

func TestEvalVar_ExplicitTypeAgreesWithDefault(t *testing.T) {
	env := NewEnv()
	reporter := diagnostics.New("")

	decl := parseSingleVarDecl(t, `var instance_type is string = "t2.micro"`)
	evalVar(decl, env, reporter)
	cfg, ok := env.Registry.Vars.lookup("instance_type")

	if reporter.HasErrors() || !ok {
		t.Fatalf("unexpected errors: %+v", reporter.Diagnostics())
	}
	if cfg.Type != "string" || cfg.Default.Str != "t2.micro" {
		t.Errorf("cfg = %+v", cfg)
	}
}

func TestEvalVar_ConflictingTypeAndDefaultIsHardError(t *testing.T) {
	env := NewEnv()
	reporter := diagnostics.New("")

	decl := parseSingleVarDecl(t, `var port is string = 123`)
	evalVar(decl, env, reporter)
	cfg, ok := env.Registry.Vars.lookup("port")

	if !reporter.HasErrors() && !ok {
		t.Fatal("expected a type-mismatch error for `is string = 123`")
	}
	if cfg != nil {
		t.Error("expected nil config when type/default conflict")
	}
}

func TestEvalVar_UnknownTypeIsError(t *testing.T) {
	env := NewEnv()
	reporter := diagnostics.New("")

	decl := parseSingleVarDecl(t, `var x is not_a_real_type`)
	evalVar(decl, env, reporter)
	cfg, _ := env.Registry.Vars.lookup("x")

	if !reporter.HasErrors() || cfg != nil {
		t.Fatal("expected an error for an unknown type name")
	}
}

func TestEvalVar_BodyFormWithDescription(t *testing.T) {
	env := NewEnv()
	reporter := diagnostics.New("")

	decl := parseSingleVarDecl(t, `var bucket_name is string {
  description = "My variable used to set bucket name"
  default     = "my_default_bucket_name"
}`)
	evalVar(decl, env, reporter)
	cfg, _ := env.Registry.Vars.lookup("bucket_name")

	if reporter.HasErrors() {
		t.Fatalf("unexpected errors: %+v", reporter.Diagnostics())
	}
	if cfg.Description != "My variable used to set bucket name" {
		t.Errorf("Description = %q", cfg.Description)
	}
	if !cfg.HasDefault || cfg.Default.Str != "my_default_bucket_name" {
		t.Errorf("Default = %+v", cfg.Default)
	}
}

func TestEvalVar_BodyFormConflictingDefaultType(t *testing.T) {
	env := NewEnv()
	reporter := diagnostics.New("")

	decl := parseSingleVarDecl(t, `var port is string {
  default = 123
}`)
	evalVar(decl, env, reporter)
	cfg, _ := env.Registry.Vars.lookup("port")

	if !reporter.HasErrors() || cfg != nil {
		t.Fatal("expected a type-mismatch error in body form too")
	}
}

// --- Flat namespace: var shares uniqueness with resource/lookup ---

func TestEvalVar_NameCollidesWithResource(t *testing.T) {
	env := NewEnv()
	reporter := diagnostics.New("")

	resourceBlock := parseBlock(t, `resource aws_instance as app_server {
  ami = "ami-123"
}`)
	evalResource(resourceBlock, env, reporter)

	varDecl := parseSingleVarDecl(t, `var app_server = "placeholder"`)
	 evalVar(varDecl, env, reporter)
	
  cfg, _ := env.Registry.Vars.lookup("app_server")

	if !reporter.HasErrors() || cfg != nil {
		t.Fatal("expected a duplicate-name error: app_server already used by a resource")
	}
}


// --- The actual design goal: references stay deferred, never inlined ---

func TestEval_VarReferenceIsAlwaysDeferred(t *testing.T) {
	env := NewEnv()
	reporter := diagnostics.New("")

	decl := parseSingleVarDecl(t, `var bucket_name = "my_default_bucket_name"`)
	evalVar(decl, env, reporter)
	if reporter.HasErrors() {
		t.Fatalf("unexpected errors declaring var: %+v", reporter.Diagnostics())
	}

	v, ok := eval(&parser.Identifier{Name: "bucket_name"}, env, reporter)
	if !ok {
		t.Fatal("expected reference to resolve")
	}
	if v.Kind != KindRef {
		t.Fatalf("Kind = %v, want KindRef — must NOT inline the default", v.Kind)
	}
	if v.Str != "var.bucket_name" {
		t.Errorf("Str = %q, want var.bucket_name", v.Str)
	}
}

func TestEval_VarReferenceDeferredEvenWithNoDefault(t *testing.T) {
	env := NewEnv()
	reporter := diagnostics.New("")

	decl := parseSingleVarDecl(t, `var instance_name`)
	evalVar(decl, env, reporter)

	v, ok := eval(&parser.Identifier{Name: "instance_name"}, env, reporter)
	if !ok || v.Kind != KindRef || v.Str != "var.instance_name" {
		t.Fatalf("got %+v, ok=%v, want KindRef var.instance_name", v, ok)
	}
}

func TestEval_UndefinedVarReferenceFails(t *testing.T) {
	env := NewEnv()
	reporter := diagnostics.New("")

	_, ok := eval(&parser.Identifier{Name: "nonexistent"}, env, reporter)
	if ok || !reporter.HasErrors() {
		t.Fatal("expected an undefined-reference error")
	}
}

// --- Integration: the exact bug just fixed in the CLI dispatch loop ---

func TestEvalResource_ReferencesVarDefault(t *testing.T) {
	env := NewEnv()
	reporter := diagnostics.New("")

	varDecl := parseSingleVarDecl(t, `var bucket_name = "my_default_bucket_name"`)
	evalVar(varDecl, env, reporter)

	resourceBlock := parseBlock(t, `resource aws_s3_bucket as my_bucket {
  bucket = bucket_name
}`)
	evalResource(resourceBlock, env, reporter)
  resCfg, _ := env.Registry.Resources.lookup("my_bucket")
	if reporter.HasErrors() {
		t.Fatalf("unexpected errors: %+v", reporter.Diagnostics())
	}
	bucket := resCfg.Extra["bucket"]
	if bucket.Kind != KindRef || bucket.Str != "var.bucket_name" {
		t.Errorf("bucket = %+v, want KindRef var.bucket_name", bucket)
	}
}

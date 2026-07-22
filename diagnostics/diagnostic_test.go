package diagnostics

import (
	"strings"
	"testing"
)

func TestFormatterFormatsSourceLocation(t *testing.T) {
	reporter := New("let answer = @;")
	reporter.ErrorAtOffset(13, "unexpected character")
	formatter := NewFormatter("let answer = @;")
	formatter.SetFilename("sample.icl")

	got := formatter.Format(reporter.Diagnostics()[0])
	want := "error: unexpected character\n" +
		"   --> sample.icl:1:14\n" +
		"   |\n" +
		" 1 | let answer = @;\n" +
		"   |              ^\n"
	if got != want {
		t.Fatalf("formatted diagnostic mismatch:\ngot:\n%q\nwant:\n%q", got, want)
	}
}

func TestFormatterFormatsCodeAndHint(t *testing.T) {
	reporter := New("print(\"hello")
	reporter.ErrorAtOffsetWithCode(6, "E0002", "unterminated string literal", "add a closing quote before the end of the string")
	formatter := NewFormatter("print(\"hello")
	formatter.SetFilename("broken.icl")

	got := formatter.Format(reporter.Diagnostics()[0])
	want := "error[E0002]: unterminated string literal\n" +
		"   --> broken.icl:1:7\n" +
		"   |\n" +
		" 1 | print(\"hello\n" +
		"   |       ^\n" +
		"   = help: add a closing quote before the end of the string\n"
	if got != want {
		t.Fatalf("formatted diagnostic mismatch:\ngot:\n%q\nwant:\n%q", got, want)
	}
}

func TestFormatterFormatsColorWhenEnabled(t *testing.T) {
	reporter := New("let answer = @;")
	reporter.ErrorAtOffsetWithCode(13, "E0001", "unexpected character \"@\"", "remove it or replace it with a valid token")
	formatter := NewFormatter("let answer = @;")
	formatter.SetColor(true)

	got := formatter.Format(reporter.Diagnostics()[0])
	want := "\x1b[1;31merror\x1b[0m[E0001]: unexpected character \"@\"\n" +
		"   --> <source>:1:14\n" +
		"   |\n" +
		" 1 | let answer = @;\n" +
		"   |              \x1b[1;31m^\x1b[0m\n" +
		"   = \x1b[36mhelp\x1b[0m: remove it or replace it with a valid token\n"
	if got != want {
		t.Fatalf("colored diagnostic mismatch:\ngot:\n%q\nwant:\n%q", got, want)
	}
}

func TestFormatterFormatsSpanRange(t *testing.T) {
	reporter := New("print(hello)")
	reporter.ReportDiagnostic(Diagnostic{
		Severity: Error,
		Code:     "E0100",
		Message:  "expected string literal",
		Span: Span{
			Start: Location{Line: 1, Column: 7, Offset: 6},
			End:   Location{Line: 1, Column: 12, Offset: 11},
		},
	})
	formatter := NewFormatter("print(hello)")

	got := formatter.Format(reporter.Diagnostics()[0])
	if !strings.Contains(got, "   |       ^^^^^\n") {
		t.Fatalf("expected range marker, got %q", got)
	}
}


func TestWriteAllWritesFormattedDiagnostics(t *testing.T) {
	reporter := New("@")
	reporter.ErrorAtOffsetWithCode(0, "E0001", "unexpected character \"@\"", "remove it or replace it with a valid token")
	formatter := NewFormatter("@")
	formatter.SetFilename("main.icl")

	var out strings.Builder
	if err := WriteAll(&out, formatter, reporter.Diagnostics()); err != nil {
		t.Fatalf("WriteAll returned error: %v", err)
	}
	if !strings.Contains(out.String(), "error[E0001]: unexpected character \"@\"") {
		t.Fatalf("expected formatted diagnostic, got %q", out.String())
	}
}

func TestReporterRecordsDiagnosticsWithoutPrinting(t *testing.T) {
	reporter := New("first\nsecond")

	reporter.WarningAtOffset(8, "check this")
	reporter.ErrorWithCode(3, 50, "E9999", "bad thing", "fix it")

	diagnostics := reporter.Diagnostics()
	if len(diagnostics) != 2 {
		t.Fatalf("got %d diagnostics, want 2", len(diagnostics))
	}
	if diagnostics[0].Severity != Warning || diagnostics[0].Span.Start.Line != 2 || diagnostics[0].Span.Start.Column != 3 || diagnostics[0].Span.Start.Offset != 8 || diagnostics[0].Message != "check this" {
		t.Fatalf("unexpected first diagnostic: %#v", diagnostics[0])
	}
	if diagnostics[1].Code != "E9999" || diagnostics[1].Help != "fix it" {
		t.Fatalf("unexpected second diagnostic: %#v", diagnostics[1])
	}
	if !reporter.HasErrors() {
		t.Fatal("expected reporter to have errors")
	}

	diagnostics[0].Message = "changed"
	if reporter.Diagnostics()[0].Message == "changed" {
		t.Fatal("Diagnostics returned mutable reporter state")
	}
}

func TestSeverityString(t *testing.T) {
	if Error.String() != "error" || Warning.String() != "warning" || Note.String() != "note" {
		t.Fatalf("unexpected severity names: %q %q %q", Error, Warning, Note)
	}
}

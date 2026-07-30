package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/aliamerj/icl/diagnostics"
	"github.com/aliamerj/icl/lexer"
	"github.com/aliamerj/icl/parser"
)

func runFmt(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fmt", flag.ContinueOnError)
	fs.SetOutput(stderr)

	write := fs.Bool("w", false, "write result to the source file instead of stdout")

	args = normalizeArgs(args)

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "usage: icl fmt <file.ic|directory> [-w]")
		return 2
	}

	path := fs.Arg(0)

	info, err := os.Stat(path)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	if !info.IsDir() {
		if err := validateICFile(path); err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return 2
		}

		return formatFile(path, *write, stdout, stderr)
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		fmt.Fprintf(stderr, "error: cannot read directory %s: %v\n", path, err)
		return 1
	}

	exitCode := 0

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if filepath.Ext(entry.Name()) != ".ic" {
			continue
		}

		file := filepath.Join(path, entry.Name())

		if code := formatFile(file, *write, stdout, stderr); code != 0 {
			exitCode = code
		}
	}

	return exitCode
}

func formatFile(path string, write bool, stdout, stderr io.Writer) int {
	source, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(stderr, "error: cannot read %s: %v\n", path, err)
		return 1
	}
	sourceText := string(source)
	reporter := diagnostics.New(sourceText)

	l := lexer.New(sourceText, reporter)
	if reporter.HasErrors() {
		printDiagnostics(reporter.Diagnostics(), string(source), path, stderr)
		return 1
	}
	p := parser.New(l.Tokens(), reporter)
	prog := p.ParseProgram()
	if reporter.HasErrors() {
		printDiagnostics(reporter.Diagnostics(), string(source), path, stderr)
		return 1
	}

	parser.AttachComments(prog, l.Comments())
	formatted := format(prog)

	if write {
		if err := os.WriteFile(path, []byte(formatted), 0644); err != nil {
			fmt.Fprintf(stderr, "error: cannot write %s: %v\n", path, err)
			return 1
		}
		return 0
	}

	fmt.Fprint(stdout, formatted)
	return 0
}

// Format reprints a Program canonically. Comments must already be attached
// to the AST by the caller; nested object and list literal trivia is still
// intentionally dropped.
func format(prog *parser.Program) string {
	var b strings.Builder
	for i, stmt := range prog.Statements {
		if i > 0 {
			b.WriteString("\n")
		}
		formatStatement(&b, stmt, 0)
	}
	return b.String()
}

func formatStatement(b *strings.Builder, stmt parser.Statement, depth int) {
	switch s := stmt.(type) {
	case *parser.Block:
		formatBlock(b, s, depth)
	case *parser.Attribute:
		formatAttribute(b, s, depth)
	}
}

func formatBlock(b *strings.Builder, block *parser.Block, depth int) {
	indent := strings.Repeat("  ", depth)
	for _, c := range block.LeadingComments {
		fmt.Fprintf(b, "%s# %s\n", indent, c)
	}
	b.WriteString(indent)
	b.WriteString(strings.ToLower(block.Keyword.String()))
	for _, label := range block.Labels {
		b.WriteString(" ")
		b.WriteString(label.Name)
	}
	if block.Name != nil {
		b.WriteString(" as ")
		b.WriteString(block.Name.Name)
	}
	b.WriteString(" {\n")

	for _, stmt := range block.Body.Statements {
		formatStatement(b, stmt, depth+1)
	}

	b.WriteString(indent)
	b.WriteString("}")
	if block.TrailingComment != "" {
		fmt.Fprintf(b, " # %s", block.TrailingComment)
	}
	b.WriteString("\n")
}

func formatAttribute(b *strings.Builder, attr *parser.Attribute, depth int) {
	indent := strings.Repeat("  ", depth)
	for _, c := range attr.LeadingComments {
		fmt.Fprintf(b, "%s# %s\n", indent, c)
	}
	fmt.Fprintf(b, "%s%s = %s", indent, attr.Name.Name, formatExpr(attr.Value))
	if attr.TrailingComment != "" {
		fmt.Fprintf(b, " # %s", attr.TrailingComment)
	}
	b.WriteString("\n")
}

func formatExpr(expr parser.Expression) string {
	switch e := expr.(type) {
	case *parser.StringLiteral:
		return fmt.Sprintf("%q", e.Value)
	case *parser.IntLiteral:
		return fmt.Sprintf("%d", e.Value)
	case *parser.FloatLiteral:
		return fmt.Sprintf("%g", e.Value)
	case *parser.BoolLiteral:
		return fmt.Sprintf("%t", e.Value)
	case *parser.Identifier:
		return e.Name
	case *parser.BinaryExpr:
		return fmt.Sprintf("%s %s %s", formatExpr(e.Left), e.Operator, formatExpr(e.Right))
	case *parser.MemberExpr:
		return fmt.Sprintf("%s.%s", formatExpr(e.Object), e.Property)
	case *parser.ListExpr:
		parts := make([]string, 0, len(e.Elements))
		for _, elem := range e.Elements {
			parts = append(parts, formatExpr(elem))
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case *parser.ObjectExpr:
		parts := make([]string, 0, len(e.Fields))
		for _, field := range e.Fields {
			parts = append(parts, formatInlineAttribute(field))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	default:
		return "?"
	}
}

func formatInlineAttribute(attr *parser.Attribute) string {
	if attr == nil || attr.Name == nil {
		return ""
	}
	return fmt.Sprintf("%s = %s", attr.Name.Name, formatExpr(attr.Value))
}

package parser

import (
	"reflect"
	"testing"

	"github.com/aliamerj/icl/diagnostics"
	"github.com/aliamerj/icl/lexer"
)

func TestAttachComments(t *testing.T) {
	src := `
# provider comment
provider aws {
  # region comment
  region = "us-east-1" # trailing region
} # end provider
`

	scan := lexer.New(src, diagnostics.New(src))
	reporter := diagnostics.New(src)
	p := New(scan.Tokens(), reporter)
	prog := p.ParseProgram()
	if reporter.HasErrors() {
		t.Fatalf("unexpected parse diagnostics: %+v", reporter.Diagnostics())
	}

	AttachComments(prog, scan.Comments())

	block := prog.Statements[0].(*Block)
	if !reflect.DeepEqual(block.LeadingComments, []string{"provider comment"}) {
		t.Fatalf("block leading comments = %#v", block.LeadingComments)
	}
	if block.TrailingComment != "end provider" {
		t.Fatalf("block trailing comment = %q", block.TrailingComment)
	}

	attr := block.Body.Statements[0].(*Attribute)
	if !reflect.DeepEqual(attr.LeadingComments, []string{"region comment"}) {
		t.Fatalf("attribute leading comments = %#v", attr.LeadingComments)
	}
	if attr.TrailingComment != "trailing region" {
		t.Fatalf("attribute trailing comment = %q", attr.TrailingComment)
	}
}

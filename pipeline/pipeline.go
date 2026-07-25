package pipeline

import (
	"github.com/aliamerj/icl/diagnostics"
	"github.com/aliamerj/icl/eval"
	"github.com/aliamerj/icl/lexer"
	"github.com/aliamerj/icl/parser"
)

func Run(source string) (*eval.Config, []diagnostics.Diagnostic) {
	// --- Lex ---
	reporter := diagnostics.New(source)
	scan := lexer.New(source, reporter)
	tokens := scan.Tokens()
	if reporter.HasErrors() {
		return nil, reporter.Diagnostics()
	}

	// --- Parse ---
	p := parser.New(tokens, reporter) 
	prog := p.ParseProgram()
	if reporter.HasErrors() {
		return nil, reporter.Diagnostics()
	}

	// --- Eval ---
	out := eval.Run(prog, reporter)
	if reporter.HasErrors() {
		return nil, reporter.Diagnostics()
	}

	return out, nil
}

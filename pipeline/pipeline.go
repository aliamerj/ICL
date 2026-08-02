package pipeline

import (
	"github.com/aliamerj/icl/diagnostics"
	"github.com/aliamerj/icl/eval"
	"github.com/aliamerj/icl/lexer"
	"github.com/aliamerj/icl/parser"
	"github.com/aliamerj/icl/tokens"
)

func Run(source string) (*eval.Environment, []diagnostics.Diagnostic) {
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
	env := eval.NewEnv()
	env.SetForwardLookup(func(name string) (string, bool) {
		return scanForLaterDeclaration(prog, name)
	})
	eval.Run(env, prog, reporter)
	if reporter.HasErrors() {
		return env, reporter.Diagnostics()
	}

	return env, nil
}

func scanForLaterDeclaration(prog *parser.Program, name string) (kind string, found bool) {
	for _, stmt := range prog.Statements {
		switch s := stmt.(type) {
		case *parser.Block:
			if (s.Keyword == tokens.RESOURCE || s.Keyword == tokens.LOOKUP) && s.Name != nil && s.Name.Name == name {
				return s.Keyword.String(), true
			}
		case *parser.VarDecl:
			if s.Name.Name == name {
				return "var", true
			}
		}
	}
	return "", false
}

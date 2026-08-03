package eval

import (
	"github.com/aliamerj/icl/diagnostics"
	"github.com/aliamerj/icl/parser"
	"github.com/aliamerj/icl/tokens"
)

func Run(env *Environment, prog *parser.Program, reporter *diagnostics.Reporter) {
	for _, stmt := range prog.Statements {
		switch s := stmt.(type) {
		case *parser.Block:
			switch s.Keyword {
			case tokens.PROVIDER:
				evalProvider(s, env, reporter)
			case tokens.RESOURCE:
				evalResource(s, env, reporter)
			case tokens.LOOKUP:
				evalLookup(s, env, reporter)
			}
		case *parser.VarDecl:
			evalVar(s, env, reporter)
		case *parser.OutputDecl:
			evalOutput(s, env, reporter)
		}
	}
}

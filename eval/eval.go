package eval

import (
	"github.com/aliamerj/icl/diagnostics"
	"github.com/aliamerj/icl/parser"
	"github.com/aliamerj/icl/tokens"
)

func Run(env *Environment, prog *parser.Program, reporter *diagnostics.Reporter) {
	for _, stmt := range prog.Statements {
		block, ok := stmt.(*parser.Block)
		if !ok {
			continue
		}
		switch block.Keyword {
		case tokens.PROVIDER:
			evalProvider(block, env, reporter)
		case tokens.RESOURCE:
			evalResource(block, env, reporter)
		}
	}
}

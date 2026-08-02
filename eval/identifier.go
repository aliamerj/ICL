package eval

import (
	"fmt"

	"github.com/aliamerj/icl/diagnostics"
	"github.com/aliamerj/icl/parser"
)

func evalIdentifier(id *parser.Identifier, env *Environment, reporter *diagnostics.Reporter) (Value, bool) {
	// Local bindings still resolve eagerly, but declared vars must remain
	// symbolic so resource attributes can keep the Terraform reference.
	if v, ok := env.get(id.Name); ok {
		return v, true
	}

	if _, found := env.Registry.Vars.lookup(id.Name); found {
		return RefValue("var." + id.Name), true
	}

	help := ""
	if env.forwardLookup != nil {
		if kind, found := env.forwardLookup(id.Name); found {
			help = fmt.Sprintf("%q is declared later in this file (as %s) — move it earlier, or declare it before this reference", id.Name, kind)
		}
	}
	reporter.ErrorAtOffsetWithCode(id.Range().Start.Offset, diagnostics.UNDEFINED_REFERENCE,
		fmt.Sprintf("undefined reference %q", id.Name), help)
	return Value{}, false
}

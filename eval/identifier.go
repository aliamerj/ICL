package eval

import (
	"fmt"

	"github.com/aliamerj/icl/diagnostics"
	"github.com/aliamerj/icl/parser"
)

func evalIdentifier(id *parser.Identifier, env *Environment, reporter *diagnostics.Reporter) (Value, bool) {
	// Local bindings still resolve eagerly, but declared vars must remain
	// symbolic so resource attributes can keep the Terraform reference.
	v, ok := env.get(id.Name)
	if !ok {
		if _, found := env.Registry.Vars.lookup(id.Name); found {
			return RefValue("var." + id.Name), true
		}
		reporter.ErrorAtOffsetWithCode(
			id.Range().Start.Offset,
			diagnostics.UNDEFINED_REFERENCE,
			fmt.Sprintf("undefined reference %q", id.Name),
			"",
		)
		return Value{}, false
	}
	return v, true
}

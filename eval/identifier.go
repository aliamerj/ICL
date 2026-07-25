package eval

import (
	"fmt"

	"github.com/aliamerj/icl/diagnostics"
	"github.com/aliamerj/icl/parser"
)

func evalIdentifier(id *parser.Identifier, env *Environment, reporter *diagnostics.Reporter) *Value {
	v, ok := env.Get(id.Name)
	if !ok {
		reporter.ErrorAtOffsetWithCode(
			id.Range().Start.Offset,
			diagnostics.UNDEFINED_REFERENCE,
			fmt.Sprintf("undefined reference %q", id.Name),
			"",
		)
		return nil
	}
	return &v
}

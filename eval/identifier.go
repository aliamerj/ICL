package eval

import (
	"fmt"

	"github.com/aliamerj/icl/diagnostics"
	"github.com/aliamerj/icl/parser"
)

func evalIdentifier(id *parser.Identifier, env *environment, reporter *diagnostics.Reporter) *Value {
	v, ok := env.get(id.Name)
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

package eval

import (
	"fmt"

	"github.com/aliamerj/icl/diagnostics"
	"github.com/aliamerj/icl/parser"
)

type OutputConfig struct {
	Name        string
	Value       Value
	Description string
	Sensitive   bool
}

func evalOutput(decl *parser.OutputDecl, env *Environment, reporter *diagnostics.Reporter) {
	name := decl.Name.Name
	registry := env.Registry
	if registry.Outputs.has(name) {
		reporter.ErrorAtOffsetWithCode(decl.Name.Rng.Start.Offset, diagnostics.DUPLICATE_NAME,
			fmt.Sprintf("output %q is already declared", name), "")
		return
	}
	cfg := &OutputConfig{Name: name}

	switch {
	case decl.Value != nil:
		val, ok := eval(decl.Value, env, reporter)
		if !ok {
			return
		}
		cfg.Value = val

	case decl.Body != nil:
		hasValue := false
		for _, stmt := range decl.Body.Statements {
			attr, ok := stmt.(*parser.Attribute)
			if !ok {
				continue
			}
			switch attr.Name.Name {
			case "value":
				val, ok := eval(attr.Value, env, reporter)
				if !ok {
					continue
				}
				cfg.Value, hasValue = val, true
			case "description":
				val, ok := eval(attr.Value, env, reporter)
				if !ok || val.Kind != KindString {
					reporter.ErrorAtOffsetWithCode(attr.Value.Range().Start.Offset, diagnostics.TYPE_MISMATCH,
						"\"description\" must be a string", "")
					continue
				}
				cfg.Description = val.Str
			case "sensitive":
				val, ok := eval(attr.Value, env, reporter)
				if !ok || val.Kind != KindBool {
					reporter.ErrorAtOffsetWithCode(attr.Value.Range().Start.Offset, diagnostics.TYPE_MISMATCH,
						"\"sensitive\" must be a bool", "")
					continue
				}
				cfg.Sensitive = val.Bool
			default:
				reporter.ErrorAtOffsetWithCode(attr.Name.Rng.Start.Offset, diagnostics.UNKNOWN_ATTRIBUTE,
					fmt.Sprintf("unknown output attribute %q", attr.Name.Name),
					"expected: value, description, sensitive")
			}
		}
		if !hasValue {
			reporter.ErrorAtOffsetWithCode(decl.Rng.Start.Offset, diagnostics.MISSING_OUTPUT_VALUE,
				fmt.Sprintf("output %q must set \"value\"", name), "")
			return
		}
	}
	registry.Outputs.add(cfg)
}

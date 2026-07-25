package eval

import (
	"fmt"

	"github.com/aliamerj/icl/diagnostics"
	"github.com/aliamerj/icl/parser"
)

// ProviderConfig is the resolved, typed result of evaluating a
// `provider` block. This is intentionally NOT ast.Block -- the AST
// is syntax, this is meaning. Keeping them separate means the AST
// never has to know what a "valid provider" looks like.
type ProviderConfig struct {
	Alias   string // e.g. "east" — from `as`, empty if none
	Name    string // e.g. "aws" - from the block's label
	Source  string
	Version string
	Extra   map[string]Value // anything else the user set, kept for forward-compat
}

func evalProvider(block *parser.Block, env *environment, reporter *diagnostics.Reporter) *ProviderConfig {
	if len(block.Labels) != 1 {
		reporter.ErrorAtOffsetWithCode(
			block.Rng.Start.Offset,
			diagnostics.INVALID_PROVIDER_BLOCK,
			"provider block must have exactly one label",
			"e.g. `provider aws { ... }`",
		)
		return nil
	}

	cfg := &ProviderConfig{
		Name:  block.Labels[0].Name,
		Extra: map[string]Value{},
	}
	if block.Name != nil {
		cfg.Alias = block.Name.Name
	}

	for _, stmt := range block.Body.Statements {
		attr, ok := stmt.(*parser.Attribute)
		if !ok {
			continue
		}

		val, ok := eval(attr.Value, env, reporter)
		if !ok {
			continue
		}

		switch attr.Name.Name {
		case "source":
			if val.Kind != KindString {
				reporter.ErrorAtOffsetWithCode(
					attr.Value.Range().Start.Offset,
					diagnostics.TYPE_MISMATCH,
					fmt.Sprintf("\"source\" must be a string, got %s", val.Kind),
					"e.g. source = \"hashicorp/aws\"",
				)
				continue
			}
			cfg.Source = val.Str
		case "version":
			if val.Kind != KindString {
				reporter.ErrorAtOffsetWithCode(
					attr.Value.Range().Start.Offset,
					diagnostics.TYPE_MISMATCH,
					fmt.Sprintf("\"version\" must be a string, got %s", val.Kind),
					"e.g. version = \"5.37.0\"",
				)
				continue
			}
			cfg.Version = val.Str
		default:
			cfg.Extra[attr.Name.Name] = val
		}
	}

	if cfg.Source == "" {
		reporter.ErrorAtOffsetWithCode(
			block.Rng.Start.Offset,
			diagnostics.MISSING_REQUIRED_FIELD,
			fmt.Sprintf("provider %q is missing required field \"source\"", cfg.Name),
			"add source = \"...\" to this block",
		)
	}

	return cfg
}

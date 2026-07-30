package eval

import (
	"fmt"

	"github.com/aliamerj/icl/diagnostics"
	"github.com/aliamerj/icl/parser"
)

// ResourceConfig is the resolved, typed result of evaluating a `resource` block.
type ResourceConfig struct {
	Type     string // e.g. "aws_instance"
	Name     string // e.g. "app_server" — from `as`
	Provider string // resolved "type" or "type.alias", empty if not specified
	Extra    map[string]Value
}

func evalResource(block *parser.Block, env *Environment, reporter *diagnostics.Reporter) {
	if len(block.Labels) != 1 {
		reporter.ErrorAtOffsetWithCode(block.Rng.Start.Offset, diagnostics.INVALID_RESOURCE_BLOCK,
			"resource block must have exactly one type label", "e.g. `resource aws_instance as name { ... }`")
		return
	}

	if block.Name == nil {
		// parser already guarantees this today (requireName=true), but
		// don't silently trust that invariant forever, assert it here too.
		reporter.ErrorAtOffsetWithCode(block.Rng.Start.Offset, diagnostics.INVALID_RESOURCE_BLOCK,
			"resource block must have a name", "")
		return
	}

	name := block.Name.Name
	if env.Registry.Resources.has(name) {
		reporter.ErrorAtOffsetWithCode(block.Name.Rng.Start.Offset, diagnostics.DUPLICATE_NAME,
			fmt.Sprintf("%q is already declared", name),
			"every resource, lookup, and input must have a unique name across the whole file")
		return
	}

	cfg := &ResourceConfig{
		Type:  block.Labels[0].Name,
		Name:  block.Name.Name,
		Extra: map[string]Value{},
	}

	for _, stmt := range block.Body.Statements {
		attr, ok := stmt.(*parser.Attribute)
		if !ok {
			continue
		}
		if attr.Name.Name == "provider" {
			typ, alias, ok := env.Registry.Providers.resolveRef(attr.Value, reporter)
			if !ok {
				continue
			}
			cfg.Provider = displayRef(typ, alias)
			continue
		}
		val, ok := eval(attr.Value, env, reporter)
		if !ok {
			continue
		}
		cfg.Extra[attr.Name.Name] = val

	}
	env.Registry.Resources.add(cfg)
}

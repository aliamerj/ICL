package eval

import (
	"fmt"

	"github.com/aliamerj/icl/diagnostics"
	"github.com/aliamerj/icl/parser"
)

type VarConfig struct {
	Name        string
	Type        string // "string" | "number" | "bool" | "list" | "object" | "any" — v0.1 set, no list(string) yet
	Default     Value
	HasDefault  bool
	Description string
	Extra       map[string]Value
}

var validVarTypes = map[string]bool{"string": true, "number": true, "bool": true, "list": true, "object": true, "any": true}

func evalVar(decl *parser.VarDecl, env *Environment, reporter *diagnostics.Reporter) {
	name := decl.Name.Name
	registry := env.Registry

	if registry.Vars.has(name) || registry.Resources.has(name) {
		reporter.ErrorAtOffsetWithCode(decl.Name.Rng.Start.Offset, diagnostics.DUPLICATE_NAME,
			fmt.Sprintf("%q is already declared", name),
			"every resource, lookup, and var must have a unique name across the whole file")
		return
	}

	declaredType := ""
	if decl.Type != nil {
		if !validVarTypes[decl.Type.Name] {
			reporter.ErrorAtOffsetWithCode(decl.Type.Rng.Start.Offset, diagnostics.INVALID_TYPE,
				fmt.Sprintf("unknown type %q", decl.Type.Name), "expected: string, number, bool, list, object, any")
			return
		}
		declaredType = decl.Type.Name
	}
	cfg := &VarConfig{
		Name:  name,
		Extra: map[string]Value{},
	}

	setDefault := func(defExpr parser.Expression) bool {
		val, ok := eval(defExpr, env, reporter) // evaluated ONCE, here, for type-checking + the tfjson literal — never at reference sites
		if !ok {
			return false
		}
		if declaredType != "" && !valueMatchesType(val, declaredType) {
			reporter.ErrorAtOffsetWithCode(defExpr.Range().Start.Offset, diagnostics.TYPE_MISMATCH,
				fmt.Sprintf("default value is %s, but declared type is %q", val.Kind, declaredType),
				fmt.Sprintf("either change `is %s` or fix the default value", declaredType))
			return false // hard error — the two clauses genuinely contradict each other
		}
		cfg.Type = declaredType
		if cfg.Type == "" {
			cfg.Type = inferTypeFromValue(val)
		}
		cfg.Default, cfg.HasDefault = val, true
		return true
	}

	switch {
	case decl.Default != nil:
		if !setDefault(decl.Default) {
			return
		}
	case decl.Body != nil:
		for _, stmt := range decl.Body.Statements {
			attr, ok := stmt.(*parser.Attribute)
			if !ok {
				continue
			}
			switch attr.Name.Name {
			case "description":
				val, ok := eval(attr.Value, env, reporter)
				if !ok {
					continue
				}
				if val.Kind != KindString {
					reporter.ErrorAtOffsetWithCode(attr.Value.Range().Start.Offset, diagnostics.TYPE_MISMATCH,
						"\"description\" must be a string", "")
					continue
				}
				cfg.Description = val.Str
			case "default":
				if !setDefault(attr.Value) {
					return
				}
			default:
				val, ok := eval(attr.Value, env, reporter)
				if !ok {
					continue
				}
				cfg.Extra[attr.Name.Name] = val
			}
		}

		if !cfg.HasDefault {
			cfg.Type = declaredType
			if cfg.Type == "" {
				cfg.Type = "any"
			}
		}
	default:
		// fully bare `var x` / `var x is string` — no default at all
		cfg.Type = declaredType
		if cfg.Type == "" {
			cfg.Type = "any"
		}
	}

	registry.Vars.add(cfg)
}

func inferTypeFromValue(v Value) string {
	switch v.Kind {
	case KindString:
		return "string"
	case KindInt, KindFloat:
		return "number"
	case KindBool:
		return "bool"
	case KindList:
		return "list"
	case KindObject:
		return "object"
	default:
		return "any"
	}
}

func valueMatchesType(v Value, typ string) bool {
	switch typ {
	case "string":
		return v.Kind == KindString
	case "number":
		return v.Kind == KindInt || v.Kind == KindFloat
	case "bool":
		return v.Kind == KindBool
	case "list":
		return v.Kind == KindList
	case "object":
		return v.Kind == KindObject
	default: // "any"
		return true
	}
}

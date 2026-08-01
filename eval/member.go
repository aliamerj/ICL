package eval

import (
	"fmt"
	"sort"
	"strings"

	"github.com/aliamerj/icl/diagnostics"
	"github.com/aliamerj/icl/parser"
)

func evalMemberExpr(m *parser.MemberExpr, env *Environment, reporter *diagnostics.Reporter) (Value, bool) {
	root, props, ok := flattenMemberChain(m)
	if !ok || len(props) == 0 {
		reporter.ErrorAtOffsetWithCode(m.Range().Start.Offset, diagnostics.INVALID_REFERENCE,
			"unrecognized reference shape", "")
		return Value{}, false
	}

	if resCfg, found := env.Registry.Resources.lookup(root); found {
		prefix := resCfg.Type + "." + resCfg.Name
		if resCfg.Kind == KindLookup {
			prefix = "data." + prefix
		}
		return RefValue(prefix + "." + strings.Join(props, ".")), true
	}

	if varCfg, found := env.Registry.Vars.lookup(root); found {
		if varCfg.Type != "object" && varCfg.Type != "any" {
			reporter.ErrorAtOffsetWithCode(m.Range().Start.Offset, diagnostics.INVALID_FIELD_ACCESS,
				fmt.Sprintf("cannot access field %q on var %q of type %q", props[0], root, varCfg.Type),
				"field access is only valid on object-typed vars")
			return Value{}, false
		}
		if !validateVarFieldPath(varCfg, props, m, reporter) {
			return Value{}, false
		}
		return RefValue("var." + root + "." + strings.Join(props, ".")), true
	}

	if len(env.Registry.Providers.instancesOf(root)) > 0 {
		switch len(props) {
		case 1:
			return resolveProviderField(root, "", props[0], m, env, reporter)
		case 2:
			return resolveProviderField(root, props[0], props[1], m, env, reporter)
		default:
			reporter.ErrorAtOffsetWithCode(m.Range().Start.Offset, diagnostics.INVALID_REFERENCE,
				fmt.Sprintf("too many segments in provider reference %q", root), "")
			return Value{}, false
		}
	}

	reporter.ErrorAtOffsetWithCode(m.Range().Start.Offset, diagnostics.UNDEFINED_REFERENCE,
		fmt.Sprintf("undefined reference %q — no resource, lookup, var, or provider with that name", root), "")
	return Value{}, false
}

func flattenMemberChain(expr parser.Expression) (root string, properties []string, ok bool) {
	switch e := expr.(type) {
	case *parser.Identifier:
		return e.Name, nil, true
	case *parser.MemberExpr:
		base, props, ok := flattenMemberChain(e.Object)
		if !ok {
			return "", nil, false
		}
		return base, append(props, e.Property), true
	default:
		return "", nil, false
	}
}

func resolveProviderField(name, alias, field string, node parser.Expression, env *Environment, reporter *diagnostics.Reporter) (Value, bool) {

	if env.Registry.Providers == nil {
		reporter.ErrorAtOffsetWithCode(node.Range().Start.Offset, diagnostics.UNDEFINED_PROVIDER,
			fmt.Sprintf("no provider configuration matches %q", displayRef(name, alias)), "")
		return Value{}, false
	}

	// If no alias was given but the type has multiple declared instances,
	// this is genuinely ambiguous — tell the user exactly which aliases exist.
	if alias == "" {
		instances := env.Registry.Providers.instancesOf(name)
		if len(instances) > 1 {
			var aliases []string
			for _, inst := range instances {
				if inst.Alias != "" {
					aliases = append(aliases, inst.Alias)
				}
			}
			reporter.ErrorAtOffsetWithCode(node.Range().Start.Offset, diagnostics.AMBIGUOUS_PROVIDER_REF,
				fmt.Sprintf("%q has multiple configurations, reference is ambiguous", name),
				fmt.Sprintf("use one of: %s", strings.Join(qualifiedNames(name, aliases), ", ")))
			return Value{}, false
		}
	}
	cfg, found := env.Registry.Providers.lookup(name, alias)
	if !found {
		reporter.ErrorAtOffsetWithCode(node.Range().Start.Offset, diagnostics.UNDEFINED_PROVIDER,
			fmt.Sprintf("no provider configuration matches %q", displayRef(name, alias)), "")
		return Value{}, false
	}
	val, ok := cfg.Extra[field]
	if !ok {
		reporter.ErrorAtOffsetWithCode(node.Range().Start.Offset, diagnostics.UNDEFINED_FIELD,
			fmt.Sprintf("provider %q has no field %q", displayRef(name, alias), field),
			fmt.Sprintf("available fields: %s", strings.Join(fieldNames(cfg.Extra), ", ")))
		return Value{}, false
	}
	return val, true

}

func qualifiedNames(typ string, aliases []string) []string {
	out := make([]string, len(aliases))
	for i, a := range aliases {
		out[i] = typ + "." + a
	}
	return out
}

func fieldNames(extra map[string]Value) []string {
	names := make([]string, 0, len(extra))
	for k := range extra {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

func validateVarFieldPath(varCfg *VarConfig, props []string, node parser.Expression, reporter *diagnostics.Reporter) bool {
	if !varCfg.HasDefault {
		return true
	}

	current := varCfg.Default
	for i, prop := range props {
		if current.Kind != KindObject {
			reporter.ErrorAtOffsetWithCode(node.Range().Start.Offset, diagnostics.INVALID_FIELD_ACCESS,
				fmt.Sprintf("cannot access field %q — %q is a %s, not an object",
					prop, strings.Join(props[:i], "."), current.Kind),
				fmt.Sprintf("var %q's default at this path is: %s", varCfg.Name, current.Kind))
			return false
		}
		next, ok := current.Object[prop]
		if !ok {
			reporter.ErrorAtOffsetWithCode(node.Range().Start.Offset, diagnostics.UNDEFINED_FIELD,
				fmt.Sprintf("var %q has no field %q", varCfg.Name, strings.Join(props[:i+1], ".")),
				fmt.Sprintf("available fields: %s", strings.Join(objectFieldNames(current.Object), ", ")))
			return false
		}
		current = next
	}
	return true
}

func objectFieldNames(obj map[string]Value) []string {
	names := make([]string, 0, len(obj))
	for k := range obj {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

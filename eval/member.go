package eval

import (
	"fmt"
	"sort"
	"strings"

	"github.com/aliamerj/icl/diagnostics"
	"github.com/aliamerj/icl/parser"
)

func evalMemberExpr(m *parser.MemberExpr, env *Environment, reporter *diagnostics.Reporter) (Value, bool) {
	// aws.region  ->  Object is a bare Identifier
	if base, ok := m.Object.(*parser.Identifier); ok {
		return resolveProviderField(base.Name, "", m.Property, m, env, reporter)
	}
	// aws.east.region  ->  Object is itself a MemberExpr (aws.east)
	if inner, ok := m.Object.(*parser.MemberExpr); ok {
		if typeIdent, ok := inner.Object.(*parser.Identifier); ok {
			return resolveProviderField(typeIdent.Name, inner.Property, m.Property, m, env, reporter)
		}
	}
	reporter.ErrorAtOffsetWithCode(m.Range().Start.Offset, diagnostics.INVALID_REFERENCE,
		"unrecognized reference shape", "")
	return Value{}, false
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

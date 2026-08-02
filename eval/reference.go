package eval

import (
	"fmt"
	"sort"
	"strings"

	"github.com/aliamerj/icl/diagnostics"
	"github.com/aliamerj/icl/parser"
)

type pathSegment struct {
	isIndex bool
	name    string // field name, if !isIndex
	index   int64  // index value, if isIndex
}

func evalReferenceChain(expr parser.Expression, env *Environment, reporter *diagnostics.Reporter) (Value, bool) {
	registry := env.Registry
	root, segs, ok := flattenChain(expr, env, reporter)
	if !ok || len(segs) == 0 {
		reporter.ErrorAtOffsetWithCode(expr.Range().Start.Offset, diagnostics.INVALID_REFERENCE,
			"unrecognized reference shape", "")
		return Value{}, false
	}

	if resCfg, found := registry.Resources.lookup(root); found {
		prefix := resCfg.Type + "." + resCfg.Name
		if resCfg.Kind == KindLookup {
			prefix = "data." + prefix
		}
		return RefValue(prefix + joinSegments(segs)), true
	}

	if varCfg, found := registry.Vars.lookup(root); found {
		if !validateVarPath(varCfg, segs, expr, reporter) {
			return Value{}, false
		}
		return RefValue("var." + root + joinSegments(segs)), true
	}

	if len(registry.Providers.instancesOf(root)) > 0 {
		// index support on provider fields not implemented yet — flagged below
		if segs[0].isIndex {
			reporter.ErrorAtOffsetWithCode(expr.Range().Start.Offset, diagnostics.INVALID_REFERENCE,
				"indexing directly into a provider is not supported", "")
			return Value{}, false
		}
		if len(segs) == 1 {
			return resolveProviderField(root, "", segs[0].name, expr, env, reporter)
		}
		if len(segs) == 2 && !segs[1].isIndex {
			return resolveProviderField(root, segs[0].name, segs[1].name, expr, env, reporter)
		}
		reporter.ErrorAtOffsetWithCode(expr.Range().Start.Offset, diagnostics.INVALID_REFERENCE,
			fmt.Sprintf("too many segments in provider reference %q", root), "")
		return Value{}, false
	}

	reporter.ErrorAtOffsetWithCode(expr.Range().Start.Offset, diagnostics.UNDEFINED_REFERENCE,
		fmt.Sprintf("undefined reference %q — no resource, lookup, var, or provider with that name", root), "")
	return Value{}, false
}

func flattenChain(expr parser.Expression, env *Environment, reporter *diagnostics.Reporter) (root string, segs []pathSegment, ok bool) {
	switch e := expr.(type) {
	case *parser.Identifier:
		return e.Name, nil, true
	case *parser.MemberExpr:
		r, s, ok := flattenChain(e.Object, env, reporter)
		if !ok {
			return "", nil, false
		}
		return r, append(s, pathSegment{name: e.Property}), true
	case *parser.IndexExpr:
		r, s, ok := flattenChain(e.Object, env, reporter)
		if !ok {
			return "", nil, false
		}
		idxVal, ok := eval(e.Index, env, reporter)
		if !ok {
			return "", nil, false
		}
		if idxVal.Kind != KindInt {
			reporter.ErrorAtOffsetWithCode(e.Index.Range().Start.Offset, diagnostics.INVALID_INDEX,
				"index must be a constant integer", "")
			return "", nil, false
		}
		return r, append(s, pathSegment{isIndex: true, index: idxVal.Int}), true
	default:
		return "", nil, false
	}
}

func joinSegments(segs []pathSegment) string {
	var b strings.Builder
	for _, s := range segs {
		if s.isIndex {
			fmt.Fprintf(&b, "[%d]", s.index)
		} else {
			b.WriteString(".")
			b.WriteString(s.name)
		}
	}
	return b.String()
}

func validateVarPath(varCfg *VarConfig, segs []pathSegment, node parser.Expression, reporter *diagnostics.Reporter) bool {
	if !varCfg.HasDefault {
		return true // nothing to check structure against
	}
	current := varCfg.Default
	pathSoFar := varCfg.Name

	for _, seg := range segs {
		if seg.isIndex {
			if current.Kind != KindList {
				reporter.ErrorAtOffsetWithCode(node.Range().Start.Offset, diagnostics.INVALID_INDEX,
					fmt.Sprintf("cannot index %q — it is a %s, not a list", pathSoFar, current.Kind), "")
				return false
			}
			if seg.index < 0 || int(seg.index) >= len(current.List) {
				reporter.ErrorAtOffsetWithCode(node.Range().Start.Offset, diagnostics.INDEX_OUT_OF_RANGE,
					fmt.Sprintf("index %d out of range for %q (length %d)", seg.index, pathSoFar, len(current.List)), "")
				return false
			}
			current = current.List[seg.index]
			pathSoFar = fmt.Sprintf("%s[%d]", pathSoFar, seg.index)
		} else {
			if current.Kind != KindObject {
				reporter.ErrorAtOffsetWithCode(node.Range().Start.Offset, diagnostics.INVALID_FIELD_ACCESS,
					fmt.Sprintf("cannot access field %q — %q is a %s, not an object", seg.name, pathSoFar, current.Kind), "")
				return false
			}
			next, ok := current.Object[seg.name]
			if !ok {
				reporter.ErrorAtOffsetWithCode(node.Range().Start.Offset, diagnostics.UNDEFINED_FIELD,
					fmt.Sprintf("%q has no field %q", pathSoFar, seg.name), "")
				return false
			}
			current = next
			pathSoFar += "." + seg.name
		}
	}
	return true
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

func fieldNames(extra map[string]Value) []string {
	names := make([]string, 0, len(extra))
	for k := range extra {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

func qualifiedNames(typ string, aliases []string) []string {
	out := make([]string, len(aliases))
	for i, a := range aliases {
		out[i] = typ + "." + a
	}
	return out
}

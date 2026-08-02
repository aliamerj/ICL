package eval

import (
	"fmt"
	"sort"
	"strings"

	"github.com/aliamerj/icl/diagnostics"
	"github.com/aliamerj/icl/parser"
)

type segKind int

const (
	segField segKind = iota
	segIndexConst
	segIndexDynamic
)

type pathSegment struct {
	kind        segKind
	name        string // segField
	index       int64  // segIndexConst
	dynamicAddr string // segIndexDynamic — raw terraform address text, e.g. "var.some_var"
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
	if instances := registry.Providers.instancesOf(root); len(instances) > 0 {
		return resolveProviderChain(root, instances, segs, expr, reporter)
	}

	help := ""
	if env.forwardLookup != nil {
		if kind, found := env.forwardLookup(root); found {
			help = fmt.Sprintf("%q is declared later in this file (as %s) — move it earlier, or declare it before this reference", root, kind)
		}
	}
	reporter.ErrorAtOffsetWithCode(expr.Range().Start.Offset, diagnostics.UNDEFINED_REFERENCE,
		fmt.Sprintf("undefined reference %q — no resource, lookup, var, or provider with that name", root), help)
	return Value{}, false
}

// resolveProviderChain fully resolves a provider reference to a concrete
// literal Value — never RefValue. Provider config isn't Terraform-addressable
// (there's no `${aws.field}` in real HCL), so unlike resource/lookup/var
// references, this MUST be resolved now, not deferred.
func resolveProviderChain(root string, instances []*ProviderConfig, segs []pathSegment, node parser.Expression, reporter *diagnostics.Reporter) (Value, bool) {
	alias := ""
	fieldStart := 0

	if len(instances) > 1 {
		// multiple configs exist -> alias is mandatory, first segment must select it
		if segs[0].kind != segField {
			reporter.ErrorAtOffsetWithCode(node.Range().Start.Offset, diagnostics.AMBIGUOUS_PROVIDER_REF,
				fmt.Sprintf("%q has multiple configurations, an alias is required", root), "")
			return Value{}, false
		}
		alias = segs[0].name
		fieldStart = 1
	} else {
		alias = instances[0].Alias
	}

	if fieldStart >= len(segs) || segs[fieldStart].kind != segField {
		reporter.ErrorAtOffsetWithCode(node.Range().Start.Offset, diagnostics.INVALID_REFERENCE,
			fmt.Sprintf("expected a field name after %q", displayRef(root, alias)), "")
		return Value{}, false
	}
	fieldName := segs[fieldStart].name

	cfg, found := lookupProvider(instances, alias)
	if !found {
		reporter.ErrorAtOffsetWithCode(node.Range().Start.Offset, diagnostics.UNDEFINED_PROVIDER,
			fmt.Sprintf("no provider configuration matches %q", displayRef(root, alias)), "")
		return Value{}, false
	}
	val, ok := cfg.Extra[fieldName]
	if !ok {
		reporter.ErrorAtOffsetWithCode(node.Range().Start.Offset, diagnostics.UNDEFINED_FIELD,
			fmt.Sprintf("provider %q has no field %q", displayRef(root, alias), fieldName),
			fmt.Sprintf("available fields: %s", strings.Join(fieldNames(cfg.Extra), ", ")))
		return Value{}, false
	}

	// walk any remaining segments (nested field access, list indexing)
	// eagerly against the now-known concrete value
	pathSoFar := fmt.Sprintf("%s.%s", displayRef(root, alias), fieldName)
	current := val
	for _, seg := range segs[fieldStart+1:] {
		switch seg.kind {
		case segField:
			if current.Kind != KindObject {
				reporter.ErrorAtOffsetWithCode(node.Range().Start.Offset, diagnostics.INVALID_FIELD_ACCESS,
					fmt.Sprintf("cannot access field %q — %q is a %s, not an object", seg.name, pathSoFar, current.Kind), "")
				return Value{}, false
			}
			next, ok := current.Object[seg.name]
			if !ok {
				reporter.ErrorAtOffsetWithCode(node.Range().Start.Offset, diagnostics.UNDEFINED_FIELD,
					fmt.Sprintf("%q has no field %q", pathSoFar, seg.name), "")
				return Value{}, false
			}
			current = next
			pathSoFar += "." + seg.name
		case segIndexConst:
			if current.Kind != KindList {
				reporter.ErrorAtOffsetWithCode(node.Range().Start.Offset, diagnostics.INVALID_INDEX,
					fmt.Sprintf("cannot index %q — it is a %s, not a list", pathSoFar, current.Kind), "")
				return Value{}, false
			}
			if seg.index < 0 || int(seg.index) >= len(current.List) {
				reporter.ErrorAtOffsetWithCode(node.Range().Start.Offset, diagnostics.INDEX_OUT_OF_RANGE,
					fmt.Sprintf("index %d out of range for %q (length %d)", seg.index, pathSoFar, len(current.List)), "")
				return Value{}, false
			}
			current = current.List[seg.index]
			pathSoFar = fmt.Sprintf("%s[%d]", pathSoFar, seg.index)
		case segIndexDynamic:
			// Genuinely unsupportable, not just unimplemented: provider
			// config is inlined as a literal in the compiled JSON, with
			// no Terraform-side address to defer to — there's no such
			// thing as "${aws.field[var.idx]}" in real HCL.
			reporter.ErrorAtOffsetWithCode(node.Range().Start.Offset, diagnostics.INVALID_INDEX,
				fmt.Sprintf("cannot use a dynamic index on %q — provider values must be indexed with a constant, since they're resolved at compile time, not by Terraform", pathSoFar),
				"")
			return Value{}, false
		}
	}
	return current, true
}

func lookupProvider(instances []*ProviderConfig, alias string) (*ProviderConfig, bool) {
	for _, inst := range instances {
		if inst.Alias == alias {
			return inst, true
		}
	}
	return nil, false
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
		switch idxVal.Kind {
		case KindInt:
			return r, append(s, pathSegment{kind: segIndexConst, index: idxVal.Int}), true
		case KindRef:
			return r, append(s, pathSegment{kind: segIndexDynamic, dynamicAddr: idxVal.Str}), true

		default:
			reporter.ErrorAtOffsetWithCode(e.Index.Range().Start.Offset, diagnostics.INVALID_INDEX,
				fmt.Sprintf("index must be a constant integer or a reference, got %s", idxVal.Kind),
				"string/bool literal indices aren't supported")
			return "", nil, false
		}

	default:
		return "", nil, false
	}
}

func joinSegments(segs []pathSegment) string {
	var b strings.Builder
	for _, s := range segs {
		switch s.kind {
		case segField:
			b.WriteString(".")
			b.WriteString(s.name)
		case segIndexConst:
			fmt.Fprintf(&b, "[%d]", s.index)
		case segIndexDynamic:
			fmt.Fprintf(&b, "[%s]", s.dynamicAddr)
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
		switch seg.kind {
		case segIndexConst:
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

		case segIndexDynamic:
			if current.Kind != KindList {
				reporter.ErrorAtOffsetWithCode(node.Range().Start.Offset, diagnostics.INVALID_INDEX,
					fmt.Sprintf("cannot index %q — it is a %s, not a list", pathSoFar, current.Kind), "")
				return false
			}
			return true // can't validate further — the actual index isn't known until apply time

		case segField:
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

func fieldNames(extra map[string]Value) []string {
	names := make([]string, 0, len(extra))
	for k := range extra {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

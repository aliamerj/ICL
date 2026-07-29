package eval

import (
	"fmt"
	"sort"
	"strings"

	"github.com/aliamerj/icl/diagnostics"
	"github.com/aliamerj/icl/parser"
)

type providerRegistry struct {
	Instances map[string]*ProviderConfig // key: "name" or "name.alias"
}


func (r *providerRegistry) Add(cfg *ProviderConfig) {
	r.Instances[registryKey(cfg.Name, cfg.Alias)] = cfg
}

func registryKey(name, alias string) string {
	if alias == "" {
		return name
	}
	return name + "." + alias
}

func (r *providerRegistry) lookup(name, alias string) (*ProviderConfig, bool) {
	cfg, ok := r.Instances[registryKey(name, alias)]
	return cfg, ok
}

// InstancesOf returns every declared instance of a provider type,
// used both to detect ambiguity and to build "did you mean" hints.
func (r *providerRegistry) instancesOf(name string) []*ProviderConfig {
	var out []*ProviderConfig
	for _, cfg := range r.Instances {
		if cfg.Name == name {
			out = append(out, cfg)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Alias < out[j].Alias
	})
	return out
}

func (r *providerRegistry) resolveRef(expr parser.Expression, reporter *diagnostics.Reporter) (typ, alias string, ok bool) {
	switch e := expr.(type) {
	case *parser.Identifier:
		typ = e.Name
	case *parser.MemberExpr:
		base, isIdent := e.Object.(*parser.Identifier)
		if !isIdent {
			reporter.ErrorAtOffsetWithCode(expr.Range().Start.Offset, diagnostics.INVALID_PROVIDER_REF,
				"provider reference must look like `type.alias`", "")
			return "", "", false
		}
		typ, alias = base.Name, e.Property
	default:
		reporter.ErrorAtOffsetWithCode(expr.Range().Start.Offset, diagnostics.INVALID_PROVIDER_REF,
			"expected a provider reference like `aws` or `aws.east`", "")
		return "", "", false
	}

	if _, found := r.lookup(typ, alias); !found {
		var hint string
		if instances := r.instancesOf(typ); len(instances) > 0 {
			var aliases []string
			for _, inst := range instances {
				aliases = append(aliases, displayRef(inst.Name, inst.Alias))
			}
			hint = "declared: " + strings.Join(aliases, ", ")
		} else {
			hint = fmt.Sprintf("no `provider %s { ... }` block was declared", typ)
		}
		reporter.ErrorAtOffsetWithCode(expr.Range().Start.Offset, diagnostics.UNDEFINED_PROVIDER,
			fmt.Sprintf("no provider configuration matches %q", displayRef(typ, alias)), hint)
		return "", "", false
	}
	return typ, alias, true
}

func displayRef(typ, alias string) string {
	if alias == "" {
		return typ
	}
	return typ + "." + alias
}

package tfjson

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/aliamerj/icl/eval"
)

// Document mirrors Terraform's JSON Configuration Syntax root shape.
type Document struct {
	Terraform *TerraformBlock           `json:"terraform,omitempty"`
	Provider  map[string]any            `json:"provider,omitempty"`
	Resource  map[string]map[string]any `json:"resource,omitempty"`
	Data      map[string]map[string]any `json:"data,omitempty"`
	Variable  map[string]any            `json:"variable,omitempty"`
}

type TerraformBlock struct {
	RequiredProviders map[string]RequiredProvider `json:"required_providers,omitempty"`
}

type RequiredProvider struct {
	Source  string `json:"source,omitempty"`
	Version string `json:"version,omitempty"`
}

type providerEntry struct {
	alias string
	attrs map[string]any
}

func newDoc() *Document {
	return &Document{
		Terraform: &TerraformBlock{RequiredProviders: make(map[string]RequiredProvider)},
	}
}

// Marshal produces the actual bytes to write as main.tf.json.
func Marshal(env *eval.Environment) ([]byte, error) {
	doc := newDoc()
	if err := buildProviders(doc, env.Registry.Providers.Instances); err != nil {
		return nil, err
	}
	buildResources(doc, env.Registry.Resources.Instances)
	buildVars(doc, env.Registry.Vars.Instances)
	return json.MarshalIndent(doc, "", "  ")
}

func buildVars(doc *Document, vars map[string]*eval.VarConfig) {
	if len(vars) == 0 {
		return
	}
	doc.Variable = map[string]any{}
	for _, v := range vars {
		entry := map[string]any{"type": v.Type}
		if v.Description != "" {
			entry["description"] = v.Description
		}
		if v.HasDefault {
			entry["default"] = v.Default.Native()
		}
		doc.Variable[v.Name] = entry
	}
}

func buildProviders(doc *Document, providers map[string]*eval.ProviderConfig) error {
	if len(providers) == 0 {
		doc.Terraform = nil
		return nil
	}

	types := providerTypes(providers)
	doc.Provider = make(map[string]any, len(types))

	for _, typ := range types {
		instances := providerInstancesByType(providers, typ)
		required, entries, err := buildProviderGroup(typ, instances)
		if err != nil {
			return err
		}

		doc.Terraform.RequiredProviders[typ] = required
		doc.Provider[typ] = encodeProviderEntries(entries)
	}

	if len(doc.Terraform.RequiredProviders) == 0 {
		doc.Terraform = nil
	}
	if len(doc.Provider) == 0 {
		doc.Provider = nil
	}
	return nil
}

func buildProviderGroup(typ string, instances []*eval.ProviderConfig) (RequiredProvider, []providerEntry, error) {
	var required RequiredProvider
	var seenRequired bool
	entries := make([]providerEntry, 0, len(instances))
	aliases := make(map[string]struct{}, len(instances))

	for _, inst := range instances {
		if inst == nil {
			continue
		}

		if inst.Source != "" || inst.Version != "" {
			current := RequiredProvider{Source: inst.Source, Version: inst.Version}
			if seenRequired && current != required {
				return RequiredProvider{}, nil, fmt.Errorf("provider %q: conflicting source/version across instances", typ)
			}
			required = current
			seenRequired = true
		}

		if _, dup := aliases[inst.Alias]; dup {
			return RequiredProvider{}, nil, fmt.Errorf("provider %q: duplicate configuration for alias %q", typ, inst.Alias)
		}
		aliases[inst.Alias] = struct{}{}

		entries = append(entries, providerEntry{
			alias: inst.Alias,
			attrs: providerAttributes(inst),
		})
	}

	if !seenRequired {
		return RequiredProvider{}, nil, fmt.Errorf("provider %q: no instance declares required source/version", typ)
	}

	return required, entries, nil
}

func providerAttributes(inst *eval.ProviderConfig) map[string]any {
	attrs := make(map[string]any, len(inst.Extra)+1)
	if inst.Alias != "" {
		attrs["alias"] = inst.Alias
	}
	for k, v := range inst.Extra {
		attrs[k] = v.Native()
	}
	return attrs
}

func encodeProviderEntries(entries []providerEntry) any {
	if len(entries) == 1 && entries[0].alias == "" {
		return entries[0].attrs
	}

	out := make([]map[string]any, len(entries))
	for i, entry := range entries {
		out[i] = entry.attrs
	}
	return out
}

func providerTypes(providers map[string]*eval.ProviderConfig) []string {
	types := make(map[string]struct{}, len(providers))
	for _, cfg := range providers {
		if cfg == nil {
			continue
		}
		types[cfg.Name] = struct{}{}
	}

	out := make([]string, 0, len(types))
	for typ := range types {
		out = append(out, typ)
	}
	sort.Strings(out)
	return out
}

func providerInstancesByType(providers map[string]*eval.ProviderConfig, typ string) []*eval.ProviderConfig {
	instances := make([]*eval.ProviderConfig, 0, len(providers))
	for _, cfg := range providers {
		if cfg != nil && cfg.Name == typ {
			instances = append(instances, cfg)
		}
	}
	sort.Slice(instances, func(i, j int) bool {
		if instances[i].Alias == instances[j].Alias {
			return instances[i].Source < instances[j].Source
		}
		return instances[i].Alias < instances[j].Alias
	})
	return instances
}

func buildResources(doc *Document, resources map[string]*eval.ResourceConfig) {
	buildResourceLike(&doc.Resource, resources, eval.KindResource)
	buildResourceLike(&doc.Data, resources, eval.KindLookup)
}

func buildResourceLike(
	target *map[string]map[string]any,
	resources map[string]*eval.ResourceConfig,
	kind eval.DeclKind,
) {
	grouped := make(map[string]map[string]any)

	for _, r := range resources {
		if r == nil || r.Kind != kind {
			continue
		}

		block, ok := grouped[r.Type]
		if !ok {
			block = make(map[string]any)
			grouped[r.Type] = block
		}

		block[r.Name] = resourceAttributes(r)
	}

	if len(grouped) != 0 {
		*target = grouped
	}
}

func resourceAttributes(r *eval.ResourceConfig) map[string]any {
	attrs := make(map[string]any, len(r.Extra)+1)
	for k, v := range r.Extra {
		attrs[k] = v.Native()
	}
	if r.Provider != "" {
		attrs["provider"] = r.Provider
	}
	return attrs
}

package tfjson

import (
	"encoding/json"
	"fmt"

	"github.com/aliamerj/icl/eval"
)

// Document mirrors Terraform's JSON Configuration Syntax root shape.
type Document struct {
	Terraform *TerraformBlock `json:"terraform,omitempty"`
	Provider  map[string]any  `json:"provider,omitempty"`
}

type TerraformBlock struct {
	RequiredProviders map[string]RequiredProvider `json:"required_providers,omitempty"`
}

type RequiredProvider struct {
	Source  string `json:"source,omitempty"`
	Version string `json:"version,omitempty"`
}

// Marshal produces the actual bytes to write as main.tf.json.
func Marshal(configs eval.Config) ([]byte, error) {
	doc, err := buildDocument(configs.Provider)
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(doc, "", "  ")
}

// buildDocument assembles the Terraform-JSON document from all resolved
func buildDocument(configs []eval.ProviderConfig) (*Document, error) {
	doc := &Document{
		Terraform: &TerraformBlock{RequiredProviders: map[string]RequiredProvider{}},
		Provider:  map[string]any{},
	}

	byType := map[string][]eval.ProviderConfig{}
	for _, cfg := range configs {
		byType[cfg.Name] = append(byType[cfg.Name], cfg)
	}

	for typ, instances := range byType {
		// exactly one instance must supply source+version; error if
		// none do, or if more than one disagrees.
		var source, version string
		for _, inst := range instances {
			if inst.Source == "" && inst.Version == "" {
				continue
			}
			if source != "" && (inst.Source != source || inst.Version != version) {
				return nil, fmt.Errorf("provider %q: conflicting source/version across aliased instances", typ)
			}
			source, version = inst.Source, inst.Version
		}
		if source == "" {
			return nil, fmt.Errorf("provider %q: no instance declares required source/version", typ)
		}
		doc.Terraform.RequiredProviders[typ] = RequiredProvider{Source: source, Version: version}

		// duplicate check: same type + same alias (or both empty) twice = error
		seenAlias := map[string]bool{}
		var entries []map[string]any
		for _, inst := range instances {
			if seenAlias[inst.Alias] {
				return nil, fmt.Errorf("provider %q: duplicate configuration for alias %q", typ, inst.Alias)
			}
			seenAlias[inst.Alias] = true

			entry := map[string]any{}
			if inst.Alias != "" {
				entry["alias"] = inst.Alias
			}
			for k, v := range inst.Extra {
				entry[k] = v.Native()
			}
			entries = append(entries, entry)
		}

		if len(entries) == 1 && entries[0]["alias"] == nil {
			doc.Provider[typ] = entries[0] // single, unaliased -> plain object, matches your existing passing test
		} else {
			doc.Provider[typ] = entries // multiple, or the one instance is aliased -> array
		}
	}

	if len(doc.Terraform.RequiredProviders) == 0 {
		doc.Terraform = nil
	}
	if len(doc.Provider) == 0 {
		doc.Provider = nil
	}
	return doc, nil
}

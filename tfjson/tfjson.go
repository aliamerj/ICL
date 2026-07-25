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

// BuildDocument assembles the Terraform-JSON document from all resolved
func BuildDocument(configs []eval.ProviderConfig) (*Document, error) {
	doc := &Document{
		Terraform: &TerraformBlock{RequiredProviders: map[string]RequiredProvider{}},
		Provider:  map[string]any{},
	}

	seen := map[string]bool{}
	for _, cfg := range configs {
		if seen[cfg.Name] {
			return nil, fmt.Errorf("duplicate provider %q", cfg.Name)
		}
		seen[cfg.Name] = true

		doc.Terraform.RequiredProviders[cfg.Name] = RequiredProvider{
			Source:  cfg.Source,
			Version: cfg.Version,
		}

		// Always emit provider.<name>, even if empty — `provider "aws" {}`
		// is meaningful in real Terraform (uses default auth/config),
		// not the same as omitting the provider entirely.
		providerConfig := map[string]any{}
		for k, v := range cfg.Extra {
			providerConfig[k] = v.Native()
		}
		doc.Provider[cfg.Name] = providerConfig
	}

	if len(doc.Terraform.RequiredProviders) == 0 {
		doc.Terraform = nil
	}
	if len(doc.Provider) == 0 {
		doc.Provider = nil
	}

	return doc, nil
}

// Marshal produces the actual bytes to write as main.tf.json.
func Marshal(configs eval.Config) ([]byte, error) {
	doc, err := BuildDocument(configs.Provider)
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(doc, "", "  ")
}


package tfjson

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/aliamerj/icl/eval"
)

func TestBuildDocument_SingleProvider(t *testing.T) {
	configs := []eval.ProviderConfig{
		{
			Name:    "aws",
			Source:  "hashicorp/aws",
			Version: "5.37.0",
			Extra: map[string]eval.Value{
				"region": eval.StringValue("eu-west-1"),
			},
		},
	}

	doc, err := BuildDocument(configs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rp := doc.Terraform.RequiredProviders["aws"]
	if rp.Source != "hashicorp/aws" || rp.Version != "5.37.0" {
		t.Errorf("required_providers.aws = %+v", rp)
	}

	providerCfg, ok := doc.Provider["aws"].(map[string]any)
	if !ok {
		t.Fatalf("provider.aws is %T, want map[string]any", doc.Provider["aws"])
	}
	if providerCfg["region"] != "eu-west-1" {
		t.Errorf("provider.aws.region = %v, want eu-west-1", providerCfg["region"])
	}
	// source/version must NOT leak into the provider config block
	if _, exists := providerCfg["source"]; exists {
		t.Error("source should not appear under provider.aws")
	}
}

func TestBuildDocument_MultipleProviders(t *testing.T) {
	configs := []eval.ProviderConfig{
		{Name: "aws", Source: "hashicorp/aws", Version: "5.0", Extra: map[string]eval.Value{}},
		{Name: "google", Source: "hashicorp/google", Version: "4.0", Extra: map[string]eval.Value{}},
	}

	doc, err := BuildDocument(configs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(doc.Terraform.RequiredProviders) != 2 {
		t.Fatalf("expected 2 required_providers, got %d", len(doc.Terraform.RequiredProviders))
	}
	if len(doc.Provider) != 2 {
		t.Fatalf("expected 2 provider entries, got %d", len(doc.Provider))
	}
}

func TestBuildDocument_DuplicateProviderNameErrors(t *testing.T) {
	configs := []eval.ProviderConfig{
		{Name: "aws", Source: "hashicorp/aws", Version: "5.0", Extra: map[string]eval.Value{}},
		{Name: "aws", Source: "hashicorp/aws", Version: "5.1", Extra: map[string]eval.Value{}},
	}

	_, err := BuildDocument(configs)
	if err == nil {
		t.Fatal("expected an error for duplicate provider name")
	}
}

func TestBuildDocument_EmptyExtraStillProducesEmptyObject(t *testing.T) {
	configs := []eval.ProviderConfig{
		{Name: "aws", Source: "hashicorp/aws", Version: "5.0", Extra: map[string]eval.Value{}},
	}

	doc, err := BuildDocument(configs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	providerCfg, ok := doc.Provider["aws"].(map[string]any)
	if !ok {
		t.Fatalf("provider.aws is %T, want map[string]any", doc.Provider["aws"])
	}
	if len(providerCfg) != 0 {
		t.Errorf("expected empty object, got %+v", providerCfg)
	}
}

func TestMarshal_ProducesValidJSON(t *testing.T) {
	configs := []eval.ProviderConfig{
		{
			Name:    "aws",
			Source:  "hashicorp/aws",
			Version: "5.37.0",
			Extra: map[string]eval.Value{
				"region":     eval.StringValue("eu-west-1"),
				"maxRetries": eval.IntValue(3),
			},
		},
	}

	out, err := Marshal(eval.Config{Provider: configs})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(out, &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out)
	}

	// Walk the exact path Terraform itself would read.
	tfBlock, ok := parsed["terraform"].(map[string]any)
	if !ok {
		t.Fatal("missing top-level 'terraform' key")
	}
	reqProviders, ok := tfBlock["required_providers"].(map[string]any)
	if !ok {
		t.Fatal("missing terraform.required_providers")
	}
	awsReq, ok := reqProviders["aws"].(map[string]any)
	if !ok {
		t.Fatal("missing terraform.required_providers.aws")
	}
	if awsReq["source"] != "hashicorp/aws" || awsReq["version"] != "5.37.0" {
		t.Errorf("required_providers.aws = %+v", awsReq)
	}

	providerBlock, ok := parsed["provider"].(map[string]any)
	if !ok {
		t.Fatal("missing top-level 'provider' key")
	}
	awsProvider, ok := providerBlock["aws"].(map[string]any)
	if !ok {
		t.Fatal("missing provider.aws")
	}
	if awsProvider["region"] != "eu-west-1" {
		t.Errorf("provider.aws.region = %v, want eu-west-1", awsProvider["region"])
	}
}

func TestMarshal_NoProvidersProducesMinimalDocument(t *testing.T) {
	out, err := Marshal(eval.Config{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	trimmed := strings.TrimSpace(string(out))
	if trimmed != "{}" {
		t.Errorf("expected empty document '{}', got %s", trimmed)
	}
}

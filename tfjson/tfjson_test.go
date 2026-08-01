package tfjson

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/aliamerj/icl/eval"
)

func TestBuildProviders_SingleProvider(t *testing.T) {
	doc := newDoc()
	configs := map[string]*eval.ProviderConfig{
		"aws": {
			Name:    "aws",
			Source:  "hashicorp/aws",
			Version: "5.37.0",
			Extra: map[string]eval.Value{
				"region": eval.StringValue("eu-west-1"),
			},
		},
	}

	if err := buildProviders(doc, configs); err != nil {
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
	if _, exists := providerCfg["source"]; exists {
		t.Error("source should not appear under provider.aws")
	}
}

func TestBuildProviders_MultipleProviderTypes(t *testing.T) {
	doc := newDoc()
	configs := map[string]*eval.ProviderConfig{
		"aws":    {Name: "aws", Source: "hashicorp/aws", Version: "5.0", Extra: map[string]eval.Value{}},
		"google": {Name: "google", Source: "hashicorp/google", Version: "4.0", Extra: map[string]eval.Value{}},
	}

	if err := buildProviders(doc, configs); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(doc.Terraform.RequiredProviders) != 2 {
		t.Fatalf("expected 2 required_providers, got %d", len(doc.Terraform.RequiredProviders))
	}
	if len(doc.Provider) != 2 {
		t.Fatalf("expected 2 provider entries, got %d", len(doc.Provider))
	}
}

func TestBuildProviders_AliasedProvidersBecomeArray(t *testing.T) {
	doc := newDoc()
	configs := map[string]*eval.ProviderConfig{
		"aws-east": {
			Name:    "aws",
			Alias:   "east",
			Source:  "hashicorp/aws",
			Version: "5.0",
			Extra:   map[string]eval.Value{"region": eval.StringValue("eu-west-1")},
		},
		"aws-west": {
			Name:    "aws",
			Alias:   "west",
			Source:  "hashicorp/aws",
			Version: "5.0",
			Extra:   map[string]eval.Value{"region": eval.StringValue("us-west-2")},
		},
	}

	if err := buildProviders(doc, configs); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	awsCfg, ok := doc.Provider["aws"].([]map[string]any)
	if !ok {
		t.Fatalf("provider.aws is %T, want []map[string]any", doc.Provider["aws"])
	}
	if len(awsCfg) != 2 {
		t.Fatalf("expected 2 aliased provider blocks, got %d", len(awsCfg))
	}
	if awsCfg[0]["alias"] != "east" && awsCfg[1]["alias"] != "east" {
		t.Fatalf("expected an east alias entry, got %+v", awsCfg)
	}
}

func TestBuildProviders_RejectsDuplicateAlias(t *testing.T) {
	doc := newDoc()
	configs := map[string]*eval.ProviderConfig{
		"aws-east-1": {
			Name:    "aws",
			Alias:   "east",
			Source:  "hashicorp/aws",
			Version: "5.0",
			Extra:   map[string]eval.Value{},
		},
		"aws-east-2": {
			Name:    "aws",
			Alias:   "east",
			Source:  "hashicorp/aws",
			Version: "5.0",
			Extra:   map[string]eval.Value{},
		},
	}

	err := buildProviders(doc, configs)
	if err == nil || !strings.Contains(err.Error(), "duplicate configuration") {
		t.Fatalf("expected duplicate alias error, got %v", err)
	}
}

func TestBuildResources_SerializesNestedValues(t *testing.T) {
	doc := newDoc()
	resources := map[string]*eval.ResourceConfig{
		"app_server": {
			Type:     "aws_instance",
			Name:     "app_server",
			Provider: "aws.east",
			Extra: map[string]eval.Value{
				"ami": eval.StringValue("ami-123456"),
				"tags": eval.ObjectValue(map[string]eval.Value{
					"Name": eval.StringValue("app-server"),
				}),
				"security_groups": eval.ListValue([]eval.Value{
					eval.StringValue("sg-1"),
					eval.RefValue("aws_security_group.shared.id"),
				}),
			},
		},
		"demo_vpc": {
			Type: "aws_vpc",
			Name: "demo_vpc",
			Extra: map[string]eval.Value{
				"cidr_block": eval.StringValue("10.0.0.0/16"),
			},
		},
	}

	buildResources(doc, resources)

	appServer, ok := doc.Resource["aws_instance"]["app_server"].(map[string]any)
	if !ok {
		t.Fatalf("resource.aws_instance.app_server is %T, want map[string]any", appServer)
	}

	if got := appServer["provider"]; got != "aws.east" {
		t.Fatalf("provider = %v, want aws.east", got)
	}
	if got := appServer["ami"]; got != "ami-123456" {
		t.Fatalf("ami = %v, want ami-123456", got)
	}

	tags, ok := appServer["tags"].(map[string]any)
	if !ok {
		t.Fatalf("tags is %T, want map[string]any", appServer["tags"])
	}
	if tags["Name"] != "app-server" {
		t.Fatalf("tags.Name = %v, want app-server", tags["Name"])
	}

	securityGroups, ok := appServer["security_groups"].([]any)
	if !ok {
		t.Fatalf("security_groups is %T, want []any", appServer["security_groups"])
	}
	if securityGroups[1] != "${aws_security_group.shared.id}" {
		t.Fatalf("security_groups[1] = %v, want Terraform reference", securityGroups[1])
	}
}

func TestMarshal_ProducesResourcesAndProviders(t *testing.T) {
	env := eval.NewEnv()
	env.Registry.Providers.Instances = map[string]*eval.ProviderConfig{
		"aws": {
			Name:    "aws",
			Source:  "hashicorp/aws",
			Version: "5.37.0",
			Extra: map[string]eval.Value{
				"region":     eval.StringValue("eu-west-1"),
				"maxRetries": eval.IntValue(3),
			},
		},
	}
	env.Registry.Resources.Instances = map[string]*eval.ResourceConfig{
		"app_server": {
			Type:     "aws_instance",
			Name:     "app_server",
			Provider: "aws",
			Extra: map[string]eval.Value{
				"ami": eval.StringValue("ami-123456"),
			},
		},
	}

	out, err := Marshal(env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(out, &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out)
	}

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
		t.Fatalf("required_providers.aws = %+v", awsReq)
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
		t.Fatalf("provider.aws.region = %v, want eu-west-1", awsProvider["region"])
	}

	resourceBlock, ok := parsed["resource"].(map[string]any)
	if !ok {
		t.Fatal("missing top-level 'resource' key")
	}
	instanceType, ok := resourceBlock["aws_instance"].(map[string]any)
	if !ok {
		t.Fatal("missing resource.aws_instance")
	}
	appServer, ok := instanceType["app_server"].(map[string]any)
	if !ok {
		t.Fatal("missing resource.aws_instance.app_server")
	}
	if appServer["provider"] != "aws" {
		t.Fatalf("resource.aws_instance.app_server.provider = %v, want aws", appServer["provider"])
	}
	if appServer["ami"] != "ami-123456" {
		t.Fatalf("resource.aws_instance.app_server.ami = %v, want ami-123456", appServer["ami"])
	}
}

func TestMarshal_NoProvidersProducesMinimalDocument(t *testing.T) {
	out, err := Marshal(eval.NewEnv())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	trimmed := strings.TrimSpace(string(out))
	if trimmed != "{}" {
		t.Errorf("expected empty document '{}', got %s", trimmed)
	}
}

func TestMarshal_ResourcesOnlyOmitsTerraformAndProviderBlocks(t *testing.T) {
	env := eval.NewEnv()
	env.Registry.Resources.Instances = map[string]*eval.ResourceConfig{
		"demo_vpc": {
			Type: "aws_vpc",
			Name: "demo_vpc",
			Extra: map[string]eval.Value{
				"cidr_block": eval.StringValue("10.0.0.0/16"),
			},
		},
	}

	out, err := Marshal(env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(out, &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out)
	}
	if _, ok := parsed["terraform"]; ok {
		t.Fatal("did not expect terraform block when there are no providers")
	}
	if _, ok := parsed["provider"]; ok {
		t.Fatal("did not expect provider block when there are no providers")
	}
	if _, ok := parsed["resource"]; !ok {
		t.Fatal("expected resource block")
	}
}

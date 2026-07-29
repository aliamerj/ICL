package eval

import "testing"

func newProviderCfg(name, alias string, extra map[string]Value) *ProviderConfig {
	return &ProviderConfig{
		Name:    name,
		Alias:   alias,
		Source:  "x/x",
		Version: "1.0",
		Extra:   extra,
	}
}

func TestRegistry_LookupUnaliased(t *testing.T) {
	env := NewEnv()
	env.Registry.Providers.Add(newProviderCfg("aws", "", nil))
	cfg, ok := env.Registry.Providers.lookup("aws", "")
	if !ok || cfg.Name != "aws" {
		t.Fatalf("got %+v, ok=%v", cfg, ok)
	}
}

func TestRegistry_LookupAliased(t *testing.T) {
	env := NewEnv()
	env.Registry.Providers.Add(newProviderCfg("aws", "east", nil))
	env.Registry.Providers.Add(newProviderCfg("aws", "west", nil))
	cfg, ok := env.Registry.Providers.lookup("aws", "east")
	if !ok || cfg.Alias != "east" {
		t.Fatalf("got %+v, ok=%v", cfg, ok)
	}
	if _, ok := env.Registry.Providers.lookup("aws", ""); ok {
		t.Error("expected no un-aliased 'aws' instance to exist")
	}
}

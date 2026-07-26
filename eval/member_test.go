package eval

import (
	"testing"

	"github.com/aliamerj/icl/diagnostics"
	"github.com/aliamerj/icl/parser"
)

func TestResolveProviderRef_ValidAlias(t *testing.T) {
	env := newEnv()
	env.registry.providers.Add(newProviderCfg("aws", "east", nil))

	typ, alias, ok := env.registry.providers.ResolveProviderRef(memberExpr("aws", "east"), diagnostics.New(""))
	if !ok || typ != "aws" || alias != "east" {
		t.Fatalf("got typ=%q alias=%q ok=%v", typ, alias, ok)
	}
}

func TestResolveProviderRef_BareIdentifierNoAlias(t *testing.T) {
	env := newEnv()
	env.registry.providers.Add(newProviderCfg("aws", "", nil))
	typ, alias, ok := env.registry.providers.ResolveProviderRef(&parser.Identifier{Name: "aws"}, diagnostics.New(""))
	if !ok || typ != "aws" || alias != "" {
		t.Fatalf("got typ=%q alias=%q ok=%v", typ, alias, ok)
	}
}

func TestResolveProviderRef_UndefinedAliasReportsError(t *testing.T) {
	env := newEnv()
	env.registry.providers.Add(newProviderCfg("aws", "east", nil))
	reporter := diagnostics.New("")
	_, _, ok := env.registry.providers.ResolveProviderRef(memberExpr("aws", "wast"), reporter)
	if ok || !reporter.HasErrors() {
		t.Fatal("expected an error for undefined alias 'wast'")
	}
}

package eval

import (
	"testing"

	"github.com/aliamerj/icl/diagnostics"
	"github.com/aliamerj/icl/parser"
)

func TestResolveProviderRef_ValidAlias(t *testing.T) {
	env := NewEnv()
	env.Registry.Providers.Add(newProviderCfg("aws", "east", nil))

	typ, alias, ok := env.Registry.Providers.resolveRef(memberExpr("aws", "east"), diagnostics.New(""))
	if !ok || typ != "aws" || alias != "east" {
		t.Fatalf("got typ=%q alias=%q ok=%v", typ, alias, ok)
	}
}

func TestResolveProviderRef_BareIdentifierNoAlias(t *testing.T) {
	env := NewEnv()
	env.Registry.Providers.Add(newProviderCfg("aws", "", nil))
	typ, alias, ok := env.Registry.Providers.resolveRef(&parser.Identifier{Name: "aws"}, diagnostics.New(""))
	if !ok || typ != "aws" || alias != "" {
		t.Fatalf("got typ=%q alias=%q ok=%v", typ, alias, ok)
	}
}

func TestResolveProviderRef_UndefinedAliasReportsError(t *testing.T) {
	env := NewEnv()
	env.Registry.Providers.Add(newProviderCfg("aws", "east", nil))
	reporter := diagnostics.New("")
	_, _, ok := env.Registry.Providers.resolveRef(memberExpr("aws", "wast"), reporter)
	if ok || !reporter.HasErrors() {
		t.Fatal("expected an error for undefined alias 'wast'")
	}
}

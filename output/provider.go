package output

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aliamerj/icl/eval"
)

// FormatPretty renders a resolved provider config as a human-readable,
// ICL-shaped block — mirrors the source syntax on purpose, so what you
// see in `icl inspect` looks like what you'd write.
func providerPretty(cfg eval.ProviderConfig) string {
	var b strings.Builder
	fmt.Fprintf(&b, "provider %s {\n", cfg.Name)
	fmt.Fprintf(&b, "  source  = %q\n", cfg.Source)
	fmt.Fprintf(&b, "  version = %q\n", cfg.Version)

	for _, k := range sortedKeys(cfg.Extra) {
		fmt.Fprintf(&b, "  %s = %s\n", k, formatValue(cfg.Extra[k]))
	}
	b.WriteString("}\n")
	return b.String()
}

// --- JSON ---

type jsonProvider struct {
	Resource string         `json:"resource"`
	Name     string         `json:"name"`
	Source   string         `json:"source"`
	Version  string         `json:"version"`
	Extra    map[string]any `json:"extra,omitempty"`
}

func providerJSON(cfg eval.ProviderConfig) (string, error) {
	extra := make(map[string]any, len(cfg.Extra))
	for k, v := range cfg.Extra {
		extra[k] = toNative(v)
	}

	out := jsonProvider{
		Resource: "provider",
		Name:     cfg.Name,
		Source:   cfg.Source,
		Version:  cfg.Version,
		Extra:    extra,
	}

	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

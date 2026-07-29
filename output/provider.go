package output

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aliamerj/icl/eval"
)

// providerPretty renders a provider config as a human-readable ICL block.
func providerPretty(cfg *eval.ProviderConfig) string {
	var b strings.Builder
	b.WriteString("provider ")
	b.WriteString(cfg.Name)
	if cfg.Alias != "" {
		b.WriteString(" as ")
		b.WriteString(cfg.Alias)
	}
	b.WriteString(" {\n")
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
	Alias    string         `json:"alias,omitempty"`
	Source   string         `json:"source"`
	Version  string         `json:"version"`
	Extra    map[string]any `json:"extra,omitempty"`
}

func providerJSON(cfg *eval.ProviderConfig) (string, error) {
	extra := make(map[string]any, len(cfg.Extra))
	for k, v := range cfg.Extra {
		extra[k] = toNative(v)
	}

	out := jsonProvider{
		Resource: "provider",
		Name:     cfg.Name,
		Alias:    cfg.Alias,
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

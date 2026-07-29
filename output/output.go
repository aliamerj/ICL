package output

import (
	"fmt"
	"io"
	"sort"

	"github.com/aliamerj/icl/eval"
)

func FormatPretty(env *eval.Environment, stdout io.Writer) {
	for _, provider := range sortedProviderConfigs(env) {
		fmt.Fprint(stdout, providerPretty(provider))
	}
}

func FormatJSON(env *eval.Environment, stdout io.Writer) error {
	for _, provider := range sortedProviderConfigs(env) {
		jsonStr, err := providerJSON(provider)
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, jsonStr)
	}

	return nil
}

func sortedProviderConfigs(env *eval.Environment) []*eval.ProviderConfig {
	if env == nil || env.Registry == nil || env.Registry.Providers == nil {
		return nil
	}

	keys := make([]string, 0, len(env.Registry.Providers.Instances))
	for key := range env.Registry.Providers.Instances {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	out := make([]*eval.ProviderConfig, 0, len(keys))
	for _, key := range keys {
		out = append(out, env.Registry.Providers.Instances[key])
	}
	return out
}

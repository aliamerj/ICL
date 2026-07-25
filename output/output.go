package output

import (
	"fmt"
	"io"

	"github.com/aliamerj/icl/eval"
)

func FormatPretty(cfg eval.Config, stdout io.Writer) {

	// provider
	for _, provider := range cfg.Provider {
		fmt.Fprint(stdout, providerPretty(provider))
	}

}

func FormatJSON(cfg eval.Config, stdout io.Writer) error {

	// provider
	for _, provider := range cfg.Provider {
		jsonStr, err := providerJSON(provider)
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, jsonStr)
	}

	return nil
}

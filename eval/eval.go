package eval

import (
	"github.com/aliamerj/icl/diagnostics"
	"github.com/aliamerj/icl/parser"
	"github.com/aliamerj/icl/tokens"
)

type Config struct {
	Provider []ProviderConfig
}

func Run(prog *parser.Program, reporter *diagnostics.Reporter) *Config {
	configs := &Config{}
	env := newEnv()

	for _, stmt := range prog.Statements {
		block, ok := stmt.(*parser.Block)
		if !ok {
			continue
		}
		switch block.Keyword {
		case tokens.PROVIDER:
			cfg := evalProvider(block, env, reporter)
			if cfg != nil {
				configs.Provider = append(configs.Provider, *cfg)
			}
			//todo more Keywords
		}
	}

	return configs
}

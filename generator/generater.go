package generator

import (
	"context"
	"fmt"

	"github.com/99designs/gqlgen/api"
	"github.com/99designs/gqlgen/plugin"
	"github.com/99designs/gqlgen/plugin/modelgen"
	"github.com/TripleMint/gqlgenc/config"
)

func Generate(ctx context.Context, cfg *config.Config, option ...api.Option) error {
	var plugins []plugin.Plugin
	if cfg.Model.IsDefined() {
		plugins = append(plugins, modelgen.New())
	}
	for _, o := range option {
		o(cfg.GQLConfig, &plugins)
	}

	if err := cfg.LoadSchema(ctx); err != nil {
		return fmt.Errorf("failed to load schema: %w", err)
	}

	if err := cfg.GQLConfig.Init(); err != nil {
		return fmt.Errorf("generating core failed: %w", err)
	}

	for _, p := range plugins {
		if mut, ok := p.(plugin.ConfigMutator); ok {
			err := mut.MutateConfig(cfg.GQLConfig)
			if err != nil {
				return fmt.Errorf("%s failed: %w", p.Name(), err)
			}
		}
	}

	return nil
}

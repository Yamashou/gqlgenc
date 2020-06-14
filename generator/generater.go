package generator

import (
	"context"

	"github.com/99designs/gqlgen/api"
	"github.com/99designs/gqlgen/plugin"
	"github.com/99designs/gqlgen/plugin/modelgen"
	"github.com/Yamashou/gqlgenc/config"
	"github.com/pkg/errors"
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
		return errors.Wrap(err, "failed to load schema")
	}

	if err := cfg.GQLConfig.Init(); err != nil {
		return errors.Wrap(err, "generating core failed")
	}

	for _, p := range plugins {
		if mut, ok := p.(plugin.ConfigMutator); ok {
			err := mut.MutateConfig(cfg.GQLConfig)
			if err != nil {
				return errors.Wrap(err, p.Name())
			}
		}
	}

	return nil
}

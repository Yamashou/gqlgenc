/*
Copyright (c) 2020 gqlgen authors

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package generator

import (
	"context"

	"golang.org/x/xerrors"

	"github.com/99designs/gqlgen/api"
	"github.com/99designs/gqlgen/plugin"
	"github.com/99designs/gqlgen/plugin/modelgen"

	"github.com/Yamashou/gqlgenc/config"
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
		return xerrors.Errorf("failed to load schema: %w", err)
	}

	if err := cfg.GQLConfig.Init(); err != nil {
		return xerrors.Errorf("generating core failed: %w", err)
	}

	for _, p := range plugins {
		if mut, ok := p.(plugin.ConfigMutator); ok {
			err := mut.MutateConfig(cfg.GQLConfig)
			if err != nil {
				// return errors.Wrap(err, p.Name())
				return xerrors.Errorf("%s failed: %w", p.Name(), err)
			}
		}
	}

	return nil
}

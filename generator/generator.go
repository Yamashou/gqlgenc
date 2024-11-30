package generator

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"syscall"

	"github.com/99designs/gqlgen/api"
	"github.com/99designs/gqlgen/plugin"
	"github.com/99designs/gqlgen/plugin/federation"
	"github.com/99designs/gqlgen/plugin/modelgen"
	"github.com/Yamashou/gqlgenc/clientgenv2"
	"github.com/Yamashou/gqlgenc/config"
	"github.com/Yamashou/gqlgenc/parsequery"
	"github.com/Yamashou/gqlgenc/querydocument"
)

// mutateHook adds the "omitempty" option to optional field from input type model as defined in graphql schema
// For more info see https://github.com/99designs/gqlgen/blob/master/docs/content/recipes/modelgen-hook.md
func mutateHook(cfg *config.Config) func(b *modelgen.ModelBuild) *modelgen.ModelBuild {
	return func(build *modelgen.ModelBuild) *modelgen.ModelBuild {
		for _, model := range build.Models {
			// only handle input type model
			if schemaModel, ok := cfg.GQLConfig.Schema.Types[model.Name]; ok && (schemaModel.IsInputType() || cfg.Generate.ShouldOmitEmptyTypes()) {
				for _, field := range model.Fields {
					// find field in graphql schema
					for _, def := range schemaModel.Fields {
						if def.Name == field.Name {
							// only add 'omitempty' on optional field as defined in graphql schema
							if !def.Type.NonNull {
								field.Tag = `json:"` + field.Name + `,omitempty"`
							}

							break
						}
					}
				}
			}
		}

		return build
	}
}

func Generate(ctx context.Context, cfg *config.Config) error {
	_ = syscall.Unlink(cfg.Client.Filename)
	if cfg.Model.IsDefined() {
		_ = syscall.Unlink(cfg.Model.Filename)
	}

	if cfg.Federation.Version != 0 {
		var (
			fedPlugin plugin.Plugin
			err       error
		)

		fedPlugin, err = federation.New(cfg.Federation.Version, cfg.GQLConfig)
		if err != nil {
			return fmt.Errorf("failed to create federation plugin: %w", err)
		}

		if fed, ok := fedPlugin.(plugin.EarlySourcesInjector); ok {
			if sources, err := fed.InjectSourcesEarly(); err == nil {
				cfg.GQLConfig.Sources = append(cfg.GQLConfig.Sources, sources...)
			} else {
				return fmt.Errorf("failed to inject federation directives: %w", err)
			}
		} else if fed, ok := fedPlugin.(plugin.EarlySourceInjector); ok {
			if source := fed.InjectSourceEarly(); source != nil {
				cfg.GQLConfig.Sources = append(cfg.GQLConfig.Sources, source)
			}
		} else {
			return errors.New("failed to inject federation directives")
		}
	}

	if err := cfg.LoadSchema(ctx); err != nil {
		return fmt.Errorf("failed to load schema: %w", err)
	}

	if err := cfg.GQLConfig.Init(); err != nil {
		return fmt.Errorf("generating core failed: %w", err)
	}

	querySources, err := clientgenv2.LoadQuerySources(cfg.Query)
	if err != nil {
		return fmt.Errorf("load query sources failed: %w", err)
	}

	queryDocument, err := parsequery.ParseQueryDocuments(cfg.GQLConfig.Schema, querySources)
	if err != nil {
		return fmt.Errorf(": %w", err)
	}

	queryDocuments, err := querydocument.QueryDocumentsByOperations(cfg.GQLConfig.Schema, queryDocument.Operations)
	if err != nil {
		return fmt.Errorf(": %w", err)
	}

	var clientGen api.Option
	if cfg.Generate != nil {
		clientGen = api.AddPlugin(clientgenv2.New(cfg.Query, queryDocument, queryDocuments, cfg.Client, cfg.Generate))
	}

	// sort Implements to ensure a deterministic output
	for _, v := range cfg.GQLConfig.Schema.Implements {
		sort.Slice(v, func(i, j int) bool { return v[i].Name < v[j].Name })
	}

	var plugins []plugin.Plugin
	if cfg.Model.IsDefined() {
		p := &modelgen.Plugin{
			MutateHook: mutateHook(cfg),
			FieldHook:  modelgen.DefaultFieldMutateHook,
		}

		plugins = append(plugins, p)
	}

	clientGen(cfg.GQLConfig, &plugins)
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

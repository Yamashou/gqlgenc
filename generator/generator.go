package generator

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"syscall"

	"github.com/99designs/gqlgen/plugin"
	"github.com/99designs/gqlgen/plugin/federation"
	"github.com/99designs/gqlgen/plugin/modelgen"
	"github.com/Yamashou/gqlgenc/v3/clientgen"
	"github.com/Yamashou/gqlgenc/v3/config"
	"github.com/Yamashou/gqlgenc/v3/queryparser"
	"github.com/vektah/gqlparser/v2/ast"
)

func Generate(ctx context.Context, cfg *config.Config) error {
	_ = syscall.Unlink(cfg.GQLGencConfig.Client.Filename)
	if cfg.GQLGenConfig.Model.IsDefined() {
		_ = syscall.Unlink(cfg.GQLGenConfig.Model.Filename)
	}

	if cfg.GQLGenConfig.Federation.Version != 0 {
		var (
			fedPlugin plugin.Plugin
			err       error
		)

		fedPlugin, err = federation.New(cfg.GQLGenConfig.Federation.Version, cfg.GQLGenConfig)
		if err != nil {
			return fmt.Errorf("failed to create federation plugin: %w", err)
		}

		if fed, ok := fedPlugin.(plugin.EarlySourcesInjector); ok {
			if sources, err := fed.InjectSourcesEarly(); err == nil {
				cfg.GQLGenConfig.Sources = append(cfg.GQLGenConfig.Sources, sources...)
			} else {
				return fmt.Errorf("failed to inject federation directives: %w", err)
			}
		} else {
			return errors.New("failed to inject federation directives")
		}
	}

	if err := cfg.LoadSchema(ctx); err != nil {
		return fmt.Errorf("failed to load schema: %w", err)
	}

	if err := cfg.GQLGenConfig.Init(); err != nil {
		return fmt.Errorf("generating core failed: %w", err)
	}

	// sort Implements to ensure a deterministic output
	for _, implements := range cfg.GQLGenConfig.Schema.Implements {
		slices.SortFunc(implements, func(a, b *ast.Definition) int {
			return strings.Compare(a.Name, b.Name)
		})
	}

	querySources, err := queryparser.LoadQuerySources(cfg.GQLGencConfig.Query)
	if err != nil {
		return fmt.Errorf("load query sources failed: %w", err)
	}

	queryDocument, err := queryparser.QueryDocument(cfg.GQLGenConfig.Schema, querySources)
	if err != nil {
		return fmt.Errorf(": %w", err)
	}

	operationQueryDocuments, err := queryparser.OperationQueryDocuments(cfg.GQLGenConfig.Schema, queryDocument.Operations)
	if err != nil {
		return fmt.Errorf(": %w", err)
	}

	var modelGen plugin.Plugin
	if cfg.GQLGenConfig.Model.IsDefined() {
		usedTypes := queryparser.TypesFromQueryDocuments(cfg.GQLGenConfig.Schema, operationQueryDocuments)
		modelGen = &modelgen.Plugin{
			MutateHook: mutateHook(cfg, usedTypes),
			FieldHook:  modelgen.DefaultFieldMutateHook,
		}
	}

	var clientGen plugin.Plugin
	if cfg.GQLGencConfig.Client.IsDefined() {
		clientGen = clientgen.New(cfg, queryDocument, operationQueryDocuments)
	}

	// modelgen before clientgen because modelgen fills cfg.GQLGenConfig.Models.
	plugins := []plugin.Plugin{modelGen, clientGen}
	for _, p := range plugins {
		if mut, ok := p.(plugin.ConfigMutator); ok {
			err := mut.MutateConfig(cfg.GQLGenConfig)
			if err != nil {
				return fmt.Errorf("%s failed: %w", p.Name(), err)
			}
		}
	}

	return nil
}

func mutateHook(cfg *config.Config, usedTypes map[string]bool) func(b *modelgen.ModelBuild) *modelgen.ModelBuild {
	return func(build *modelgen.ModelBuild) *modelgen.ModelBuild {
		// only generate used models
		if cfg.GQLGencConfig.Generate != nil && cfg.GQLGencConfig.Generate.UsedOnlyModels != nil && *cfg.GQLGencConfig.Generate.UsedOnlyModels {
			var newModels []*modelgen.Object
			for _, model := range build.Models {
				if usedTypes[model.Name] {
					newModels = append(newModels, model)
				}
			}
			build.Models = newModels
			build.Interfaces = nil
		}

		return build
	}
}

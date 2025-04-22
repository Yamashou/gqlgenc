package gen

import (
	"context"
	"fmt"
	"github.com/99designs/gqlgen/codegen/templates"
	"github.com/Yamashou/gqlgenc/v3/generator"
	"github.com/Yamashou/gqlgenc/v3/modelgen"
	"github.com/Yamashou/gqlgenc/v3/querygen"
	"slices"
	"strings"
	"syscall"

	"github.com/99designs/gqlgen/plugin"
	"github.com/99designs/gqlgen/plugin/federation"
	"github.com/Yamashou/gqlgenc/v3/clientgen"
	"github.com/Yamashou/gqlgenc/v3/config"
	"github.com/Yamashou/gqlgenc/v3/queryparser"
	"github.com/vektah/gqlparser/v2/ast"
)

func Generate(ctx context.Context, cfg *config.Config) error {
	if cfg.GQLGenConfig.Model.IsDefined() {
		_ = syscall.Unlink(cfg.GQLGenConfig.Model.Filename)
	}
	if cfg.GQLGencConfig.QueryGen.IsDefined() {
		_ = syscall.Unlink(cfg.GQLGencConfig.QueryGen.Filename)
	}
	if cfg.GQLGencConfig.ClientGen.IsDefined() {
		_ = syscall.Unlink(cfg.GQLGencConfig.ClientGen.Filename)
	}

	if cfg.GQLGenConfig.Federation.Version != 0 {
		fedPlugin, err := federation.New(cfg.GQLGenConfig.Federation.Version, cfg.GQLGenConfig)
		if err != nil {
			return fmt.Errorf("failed to create federation plugin: %w", err)
		}
		if sources, err := fedPlugin.InjectSourcesEarly(); err == nil {
			cfg.GQLGenConfig.Sources = append(cfg.GQLGenConfig.Sources, sources...)
		} else {
			return fmt.Errorf("failed to inject federation directives: %w", err)
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
	if err := ValidateOperationList(queryDocument.Operations); err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	operationQueryDocuments, err := queryparser.OperationQueryDocuments(cfg.GQLGenConfig.Schema, queryDocument.Operations)
	if err != nil {
		return fmt.Errorf(": %w", err)
	}
	var modelGen plugin.Plugin
	if cfg.GQLGenConfig.Model.IsDefined() {
		modelGen = modelgen.New(cfg, operationQueryDocuments)
	}

	// modelgen before querygen and clientgen because modelgen fills cfg.GQLGenConfig.Models.
	if mut, ok := modelGen.(plugin.ConfigMutator); ok {
		err := mut.MutateConfig(cfg.GQLGenConfig)
		if err != nil {
			return fmt.Errorf("%s failed: %w", modelGen.Name(), err)
		}
	}

	// Generate code from template and document source
	sourceGenerator := generator.NewSourceGenerator(cfg)
	source := generator.NewSource(cfg.GQLGenConfig.Schema, queryDocument, sourceGenerator)

	// Fragment
	fragments, err := source.Fragments()
	if err != nil {
		return fmt.Errorf("generating fragment failed: %w", err)
	}
	for _, fragment := range fragments {
		cfg.GQLGenConfig.Models.Add(fragment.Name, fmt.Sprintf("%s.%s", cfg.GQLGencConfig.QueryGen.Pkg(), templates.ToGo(fragment.Name)))
	}

	// Operation Response
	operationResponses, err := source.OperationResponses()
	if err != nil {
		return fmt.Errorf("generating operation response failed: %w", err)
	}

	for _, operationResponse := range operationResponses {
		cfg.GQLGenConfig.Models.Add(operationResponse.Name, fmt.Sprintf("%s.%s", cfg.GQLGencConfig.QueryGen.Pkg(), templates.ToGo(operationResponse.Name)))
	}

	// Operation
	operations, err := source.Operations(operationQueryDocuments)
	if err != nil {
		return fmt.Errorf("generating operation failed: %w", err)
	}

	// Struct Source TODO: なにこれ？
	structSources := source.ResponseSubTypes()

	// Plugins
	var gqlgencPlugins []plugin.Plugin

	// querygen
	if cfg.GQLGencConfig.QueryGen.IsDefined() {
		queryGen := querygen.New(cfg, fragments, operations, operationResponses, structSources)
		gqlgencPlugins = append(gqlgencPlugins, queryGen)
	}

	// clientgen
	if cfg.GQLGencConfig.ClientGen.IsDefined() {
		clientGen := clientgen.New(cfg, operations)
		gqlgencPlugins = append(gqlgencPlugins, clientGen)
	}
	for _, gqlgencPlugin := range gqlgencPlugins {
		if mut, ok := gqlgencPlugin.(plugin.ConfigMutator); ok {
			err := mut.MutateConfig(cfg.GQLGenConfig)
			if err != nil {
				return fmt.Errorf("%s failed: %w", gqlgencPlugin.Name(), err)
			}
		}
	}

	return nil
}
func ValidateOperationList(os ast.OperationList) error {
	if err := IsUniqueName(os); err != nil {
		return fmt.Errorf("is not unique operation name: %w", err)
	}

	return nil
}
func IsUniqueName(os ast.OperationList) error {
	operationNames := make(map[string]struct{})
	for _, operation := range os {
		_, exist := operationNames[templates.ToGo(operation.Name)]
		if exist {
			return fmt.Errorf("duplicate operation: %s", operation.Name)
		}
	}

	return nil
}

package plugins

import (
	"fmt"

	"github.com/99designs/gqlgen/codegen/templates"

	"github.com/Yamashou/gqlgenc/v3/config"
	"github.com/Yamashou/gqlgenc/v3/gotype"
	"github.com/Yamashou/gqlgenc/v3/plugins/clientgen"
	"github.com/Yamashou/gqlgenc/v3/plugins/modelgen"
	"github.com/Yamashou/gqlgenc/v3/plugins/querygen"
	"github.com/Yamashou/gqlgenc/v3/queryparser"
)

func Run(cfg *config.Config) error {
	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	// Load Query
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

	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	// modelgen
	if cfg.GQLGenConfig.Model.IsDefined() {
		modelGen := modelgen.New(cfg, operationQueryDocuments)
		if err := modelGen.MutateConfig(cfg.GQLGenConfig); err != nil {
			return fmt.Errorf("%s failed: %w", modelGen.Name(), err)
		}
	}

	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	// generating source
	// must generate source after modelgen
	goTypeBinder := gotype.NewBinder(cfg, queryDocument)

	// Fragment
	fragments, err := goTypeBinder.Fragments()
	if err != nil {
		return fmt.Errorf("generating fragment failed: %w", err)
	}
	for _, fragment := range fragments {
		cfg.GQLGenConfig.Models.Add(fragment.Name, fmt.Sprintf("%s.%s", cfg.GQLGencConfig.QueryGen.Pkg(), templates.ToGo(fragment.Name)))
	}

	// Operation Response
	operationResponses, err := goTypeBinder.OperationResponses()
	if err != nil {
		return fmt.Errorf("generating operation response failed: %w", err)
	}

	for _, operationResponse := range operationResponses {
		cfg.GQLGenConfig.Models.Add(operationResponse.Name, fmt.Sprintf("%s.%s", cfg.GQLGencConfig.QueryGen.Pkg(), templates.ToGo(operationResponse.Name)))
	}

	// Operation
	operations, err := goTypeBinder.Operations(operationQueryDocuments)
	if err != nil {
		return fmt.Errorf("generating operation failed: %w", err)
	}

	// Struct Source TODO: なにこれ？
	structSources := goTypeBinder.ResponseSubTypes()

	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	// gqlgenc Plugins

	// querygen
	if cfg.GQLGencConfig.QueryGen.IsDefined() {
		queryGen := querygen.New(cfg, fragments, operations, operationResponses, structSources)
		if err := queryGen.MutateConfig(cfg.GQLGenConfig); err != nil {
			return fmt.Errorf("%s failed: %w", queryGen.Name(), err)
		}
	}

	// clientgen
	if cfg.GQLGencConfig.ClientGen.IsDefined() {
		clientGen := clientgen.New(cfg, operations)
		if err := clientGen.MutateConfig(cfg.GQLGenConfig); err != nil {
			return fmt.Errorf("%s failed: %w", clientGen.Name(), err)
		}
	}

	return nil
}

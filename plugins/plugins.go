package plugins

import (
	"fmt"
	"github.com/Yamashou/gqlgenc/v3/clientgenv2"

	"github.com/Yamashou/gqlgenc/v3/config"
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
	// テンプレートと情報ソースを元にコード生成
	// Generate code from template and document source
	sourceGenerator := clientgenv2.NewSourceGenerator(cfg)
	source := clientgenv2.NewSource(cfg.GQLGenConfig.Schema, queryDocument, sourceGenerator)

	fragments, err := source.Fragments()
	if err != nil {
		return fmt.Errorf("generating fragment failed: %w", err)
	}

	operationResponses, err := source.OperationResponses()
	if err != nil {
		return fmt.Errorf("generating operation response failed: %w", err)
	}

	operations := source.Operations(operationQueryDocuments)

	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	// gqlgenc Plugins

	// querygen
	if cfg.GQLGencConfig.QueryGen.IsDefined() {
		queryGen := querygen.New(cfg, operations, operationResponses, fragments)
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

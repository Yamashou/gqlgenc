package clientgenv2

import (
	"fmt"

	"github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/plugin"
	gqlgencConfig "github.com/Yamashou/gqlgenc/config"
	"github.com/vektah/gqlparser/v2/ast"
)

var _ plugin.ConfigMutator = &Plugin{}

type Plugin struct {
	queryFilePaths          []string
	queryDocument           *ast.QueryDocument
	operationQueryDocuments []*ast.QueryDocument
	Client                  config.PackageConfig
	GenerateConfig          *gqlgencConfig.GenerateConfig
}

func New(queryFilePaths []string, queryDocument *ast.QueryDocument, operationQueryDocuments []*ast.QueryDocument, client config.PackageConfig, generateConfig *gqlgencConfig.GenerateConfig) *Plugin {
	return &Plugin{
		queryFilePaths:          queryFilePaths,
		queryDocument:           queryDocument,
		operationQueryDocuments: operationQueryDocuments,
		Client:                  client,
		GenerateConfig:          generateConfig,
	}
}

func (p *Plugin) Name() string {
	return "clientgen"
}

func (p *Plugin) MutateConfig(cfg *config.Config) error {
	// テンプレートと情報ソースを元にコード生成
	// Generate code from template and document source
	sourceGenerator := NewSourceGenerator(cfg, p.Client)
	source := NewSource(cfg.Schema, p.queryDocument, sourceGenerator, p.GenerateConfig)

	fragments, err := source.Fragments()
	if err != nil {
		return fmt.Errorf("generating fragment failed: %w", err)
	}

	operationResponses, err := source.OperationResponses()
	if err != nil {
		return fmt.Errorf("generating operation response failed: %w", err)
	}

	operations, err := source.Operations(p.operationQueryDocuments)
	if err != nil {
		return fmt.Errorf("generating operation failed: %w", err)
	}

	if err := RenderTemplate(cfg, fragments, operations, operationResponses, source.ResponseSubTypes(), p.GenerateConfig, p.Client); err != nil {
		return fmt.Errorf("template failed: %w", err)
	}

	return nil
}

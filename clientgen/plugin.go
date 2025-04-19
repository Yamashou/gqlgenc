package clientgen

import (
	"fmt"
	gqlgenconfig "github.com/99designs/gqlgen/codegen/config"

	"github.com/99designs/gqlgen/plugin"
	"github.com/Yamashou/gqlgenc/v3/config"
	"github.com/vektah/gqlparser/v2/ast"
)

var _ plugin.ConfigMutator = &Plugin{}

type Plugin struct {
	cfg                     *config.Config
	queryDocument           *ast.QueryDocument
	operationQueryDocuments []*ast.QueryDocument
}

func New(cfg *config.Config, queryDocument *ast.QueryDocument, operationQueryDocuments []*ast.QueryDocument) *Plugin {
	return &Plugin{
		cfg:                     cfg,
		queryDocument:           queryDocument,
		operationQueryDocuments: operationQueryDocuments,
	}
}

func (p *Plugin) Name() string {
	return "clientgen"
}

func (p *Plugin) MutateConfig(gqlgenCfg *gqlgenconfig.Config) error {
	// Generate code from template and document source
	sourceGenerator := NewSourceGenerator(p.cfg)
	source := NewSource(gqlgenCfg.Schema, p.queryDocument, sourceGenerator)

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

	if err := RenderTemplate(p.cfg, fragments, operations, operationResponses, source.ResponseSubTypes()); err != nil {
		return fmt.Errorf("template failed: %w", err)
	}

	return nil
}

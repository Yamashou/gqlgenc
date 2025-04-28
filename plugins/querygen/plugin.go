package querygen

import (
	"fmt"

	gqlgenconfig "github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/plugin"
	"github.com/Yamashou/gqlgenc/v3/config"
	"github.com/Yamashou/gqlgenc/v3/gotype"
	"golang.org/x/tools/imports"
)

var _ plugin.ConfigMutator = &Plugin{}

type Plugin struct {
	cfg                *config.Config
	operations         []*gotype.Operation
	operationResponses []*gotype.OperationResponse
	queryTypes         []*gotype.QueryType
	fragments          []*gotype.Fragment
}

func New(cfg *config.Config, operations []*gotype.Operation, operationResponses []*gotype.OperationResponse, queryTypes []*gotype.QueryType, fragments []*gotype.Fragment) *Plugin {
	return &Plugin{
		cfg:                cfg,
		fragments:          fragments,
		operations:         operations,
		operationResponses: operationResponses,
		queryTypes:         queryTypes,
	}
}

func (p *Plugin) Name() string {
	return "querygen"
}

func (p *Plugin) MutateConfig(_ *gqlgenconfig.Config) error {
	if err := RenderTemplate(p.cfg, p.fragments, p.operations, p.operationResponses, p.queryTypes); err != nil {
		return fmt.Errorf("template failed: %w", err)
	}

	if _, err := imports.Process(p.cfg.GQLGencConfig.QueryGen.Filename, nil, nil); err != nil {
		return fmt.Errorf("go imports: %w", err)
	}

	return nil
}

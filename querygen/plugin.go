package querygen

import (
	"fmt"
	gqlgenconfig "github.com/99designs/gqlgen/codegen/config"
	"github.com/Yamashou/gqlgenc/v3/generator"

	"github.com/99designs/gqlgen/plugin"
	"github.com/Yamashou/gqlgenc/v3/config"
)

var _ plugin.ConfigMutator = &Plugin{}

type Plugin struct {
	cfg                *config.Config
	fragments          []*generator.Fragment
	operations         []*generator.Operation
	operationResponses []*generator.OperationResponse
	structSources      []*generator.StructSource
}

func New(cfg *config.Config, fragments []*generator.Fragment, operations []*generator.Operation, operationResponses []*generator.OperationResponse, structSources []*generator.StructSource) *Plugin {
	return &Plugin{
		cfg:                cfg,
		fragments:          fragments,
		operations:         operations,
		operationResponses: operationResponses,
		structSources:      structSources,
	}
}

func (p *Plugin) Name() string {
	return "querygen"
}

func (p *Plugin) MutateConfig(_ *gqlgenconfig.Config) error {
	if err := RenderTemplate(p.cfg, p.fragments, p.operations, p.operationResponses, p.structSources); err != nil {
		return fmt.Errorf("template failed: %w", err)
	}

	return nil
}

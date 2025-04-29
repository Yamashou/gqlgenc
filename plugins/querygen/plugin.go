package querygen

import (
	"fmt"
	"github.com/Yamashou/gqlgenc/v3/clientgenv2"
	"go/types"

	gqlgenconfig "github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/plugin"
	"github.com/Yamashou/gqlgenc/v3/config"
	"golang.org/x/tools/imports"
)

var _ plugin.ConfigMutator = &Plugin{}

type Plugin struct {
	cfg            *config.Config
	operations     []*clientgenv2.Operation
	generatedTypes []types.Type
}

func New(cfg *config.Config, operations []*clientgenv2.Operation, generatedTypes []types.Type) *Plugin {
	return &Plugin{
		cfg:            cfg,
		operations:     operations,
		generatedTypes: generatedTypes,
	}
}

func (p *Plugin) Name() string {
	return "querygen"
}

func (p *Plugin) MutateConfig(_ *gqlgenconfig.Config) error {
	if err := clientgenv2.RenderTemplate(p.cfg, p.operations, p.generatedTypes); err != nil {
		return fmt.Errorf("template failed: %w", err)
	}

	if _, err := imports.Process(p.cfg.GQLGencConfig.QueryGen.Filename, nil, nil); err != nil {
		return fmt.Errorf("go imports: %w", err)
	}

	return nil
}

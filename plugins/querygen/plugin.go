package querygen

import (
	"fmt"
	"github.com/Yamashou/gqlgenc/v3/source"
	"go/types"

	gqlgenconfig "github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/plugin"
	"github.com/Yamashou/gqlgenc/v3/config"
	"golang.org/x/tools/imports"
)

var _ plugin.ConfigMutator = &Plugin{}

type Plugin struct {
	cfg        *config.Config
	operations []*source.Operation
	goTypes    []types.Type
}

func New(cfg *config.Config, operations []*source.Operation, goTypes []types.Type) *Plugin {
	return &Plugin{
		cfg:        cfg,
		operations: operations,
		goTypes:    goTypes,
	}
}

func (p *Plugin) Name() string {
	return "querygen"
}

func (p *Plugin) MutateConfig(_ *gqlgenconfig.Config) error {
	if err := RenderTemplate(p.cfg, p.operations, p.goTypes); err != nil {
		return fmt.Errorf("template failed: %w", err)
	}

	if _, err := imports.Process(p.cfg.GQLGencConfig.QueryGen.Filename, nil, nil); err != nil {
		return fmt.Errorf("go imports: %w", err)
	}

	return nil
}

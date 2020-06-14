package clientgen

import (
	"fmt"

	"github.com/99designs/gqlgen/codegen/config"

	"github.com/99designs/gqlgen/codegen/templates"
)

func ModelRecord(cfg *config.Config, fragments []*Fragment, operationResponses []*OperationResponse, client config.PackageConfig) {
	for _, fragment := range fragments {
		name := fragment.Name
		cfg.Models.Add(
			name,
			fmt.Sprintf("%s.%s", client.Pkg(), templates.ToGo(name)),
		)
	}
	for _, operationResponse := range operationResponses {
		name := operationResponse.Name
		cfg.Models.Add(
			name,
			fmt.Sprintf("%s.%s", client.Pkg(), templates.ToGo(name)),
		)
	}
}

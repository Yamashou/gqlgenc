package clientgenv2

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vektah/gqlparser/v2/ast"
)

func TestValidateOperationList(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		operationNames []string
		wantErr        string
	}{
		{
			name:           "distinct names",
			operationNames: []string{"GetUser", "ListUsers"},
		},
		{
			name:           "exact duplicate",
			operationNames: []string{"GetUser", "GetUser"},
			wantErr:        "duplicate operation: GetUser",
		},
		{
			name:           "names that collide after conversion to Go",
			operationNames: []string{"getUser", "get_user"},
			wantErr:        "duplicate operation: get_user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			operations := make(ast.OperationList, 0, len(tt.operationNames))
			for _, name := range tt.operationNames {
				operations = append(operations, &ast.OperationDefinition{Name: name})
			}

			err := ValidateOperationList(operations)
			if tt.wantErr == "" {
				require.NoError(t, err)

				return
			}

			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

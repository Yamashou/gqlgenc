package introspection

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseIntrospectionQuery_Parse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		filename    string
		expectedErr error
	}{
		{"no mutation in schema", "testdata/introspection_result_no_mutation.json", nil},
	}

	for _, testCase := range tests {
		test := testCase
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			query := readQueryResult(t, test.filename)

			if test.expectedErr == nil {
				require.NotPanics(t, func() {
					ast := ParseIntrospectionQuery("test", query)
					require.NotNil(t, ast)
				})
			} else {
				require.PanicsWithValue(t, test.expectedErr, func() {
					ast := ParseIntrospectionQuery("test", query)
					require.Nil(t, ast)
				})
			}
		})
	}
}

func readQueryResult(t *testing.T, filename string) Query {
	t.Helper()

	data, err := os.ReadFile(filename)
	require.NoError(t, err)

	query := Query{}
	err = json.Unmarshal(data, &query)
	require.NoError(t, err)

	return query
}

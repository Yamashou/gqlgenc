package introspection

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseIntrospectionQuery_Parse(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		expectedErr error
	}{
		{"no mutation in schema", "testdata/introspection_result_no_mutation.json", nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			query := readQueryResult(t, test.filename)

			if test.expectedErr == nil {
				require.NotPanics(t, func() {
					ParseIntrospectionQuery(query)
				})
			} else {
				require.PanicsWithValue(t, test.expectedErr, func() {
					ParseIntrospectionQuery(query)
				})
			}
		})
	}
}

func readQueryResult(t *testing.T, filename string) Query {
	data, err := ioutil.ReadFile(filename)
	require.NoError(t, err)

	query := Query{}
	err = json.Unmarshal(data, &query)
	require.NoError(t, err)

	return query
}
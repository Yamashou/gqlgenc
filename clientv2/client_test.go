package clientv2

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

const (
	qqlSingleErr        = `{"errors":[{"path":["query GetUser","viewer","repositories","nsodes"],"extensions":{"code":"undefinedField","typeName":"RepositoryConnection","fieldName":"nsodes"},"locations":[{"line":6,"column":4}],"message":"Field 'nsodes' doesn't exist on type 'RepositoryConnection'"}]}`
	gqlMultipleErr      = `{"errors":[{"path":["query GetUser","viewer","repositories","nsodes"],"extensions":{"code":"undefinedField","typeName":"RepositoryConnection","fieldName":"nsodes"},"locations":[{"line":6,"column":4}],"message":"Field 'nsodes' doesn't exist on type 'RepositoryConnection'"},{"path":["query GetUser"],"extensions":{"code":"variableNotUsed","variableName":"languageFirst"},"locations":[{"line":1,"column":1}],"message":"Variable $languageFirst is declared by GetUser but not used"},{"path":["fragment LanguageFragment"],"extensions":{"code":"useAndDefineFragment","fragmentName":"LanguageFragment"},"locations":[{"line":18,"column":1}],"message":"Fragment LanguageFragment was defined, but not used"}]}`
	gqlDataAndErr       = `{"data": {"something": "some data"},"errors":[{"path":["query GetUser","viewer","repositories","nsodes"],"extensions":{"code":"undefinedField","typeName":"RepositoryConnection","fieldName":"nsodes"},"locations":[{"line":6,"column":4}],"message":"Field 'nsodes' doesn't exist on type 'RepositoryConnection'"}]}`
	invalidJSON         = "invalid"
	validData           = `{"data":{"something": "some data"}}`
	withBadDataFormat   = `{"data": "notAndObject"}`
	withBadErrorsFormat = `{"errors": "bad"}`
)

type fakeRes struct {
	Something string `json:"something"`
}

func TestUnmarshal(t *testing.T) {
	t.Parallel()
	t.Run("single error", func(t *testing.T) {
		t.Parallel()
		var path ast.Path
		_ = json.Unmarshal([]byte(`["query GetUser","viewer","repositories","nsodes"]`), &path)
		r := &fakeRes{}
		err := unmarshal([]byte(qqlSingleErr), r)
		expectedErr := &GqlErrorList{
			Errors: gqlerror.List{{
				Message: "Field 'nsodes' doesn't exist on type 'RepositoryConnection'",
				Path:    path,
				Locations: []gqlerror.Location{{
					Line:   6,
					Column: 4,
				}},
				Extensions: map[string]interface{}{
					"code":      "undefinedField",
					"typeName":  "RepositoryConnection",
					"fieldName": "nsodes",
				},
			}},
		}
		require.Equal(t, err, expectedErr)
	})

	t.Run("multiple errors", func(t *testing.T) {
		t.Parallel()
		var path1 ast.Path
		_ = json.Unmarshal([]byte(`["query GetUser","viewer","repositories","nsodes"]`), &path1)
		var path2 ast.Path
		_ = json.Unmarshal([]byte(`["query GetUser"]`), &path2)
		var path3 ast.Path
		_ = json.Unmarshal([]byte(`["fragment LanguageFragment"]`), &path3)
		r := &fakeRes{}
		err := unmarshal([]byte(gqlMultipleErr), r)
		expectedErr := &GqlErrorList{
			Errors: gqlerror.List{
				{
					Message: "Field 'nsodes' doesn't exist on type 'RepositoryConnection'",
					Path:    path1,
					Locations: []gqlerror.Location{{
						Line:   6,
						Column: 4,
					}},
					Extensions: map[string]interface{}{
						"code":      "undefinedField",
						"typeName":  "RepositoryConnection",
						"fieldName": "nsodes",
					},
				},
				{
					Message: "Variable $languageFirst is declared by GetUser but not used",
					Path:    path2,
					Locations: []gqlerror.Location{{
						Line:   1,
						Column: 1,
					}},
					Extensions: map[string]interface{}{
						"code":         "variableNotUsed",
						"variableName": "languageFirst",
					},
				},
				{
					Message: "Fragment LanguageFragment was defined, but not used",
					Path:    path3,
					Locations: []gqlerror.Location{{
						Line:   18,
						Column: 1,
					}},
					Extensions: map[string]interface{}{
						"code":         "useAndDefineFragment",
						"fragmentName": "LanguageFragment",
					},
				},
			},
		}
		require.Equal(t, err, expectedErr)
	})

	t.Run("data and error", func(t *testing.T) {
		t.Parallel()
		var path ast.Path
		_ = json.Unmarshal([]byte(`["query GetUser","viewer","repositories","nsodes"]`), &path)
		r := &fakeRes{}
		err := unmarshal([]byte(gqlDataAndErr), r)
		expectedErr := &GqlErrorList{
			Errors: gqlerror.List{{
				Message: "Field 'nsodes' doesn't exist on type 'RepositoryConnection'",
				Path:    path,
				Locations: []gqlerror.Location{{
					Line:   6,
					Column: 4,
				}},
				Extensions: map[string]interface{}{
					"code":      "undefinedField",
					"typeName":  "RepositoryConnection",
					"fieldName": "nsodes",
				},
			}},
		}
		require.Equal(t, err, expectedErr)
	})

	t.Run("invalid json", func(t *testing.T) {
		t.Parallel()
		r := &fakeRes{}
		err := unmarshal([]byte(invalidJSON), r)
		require.EqualError(t, err, "failed to decode data invalid: invalid character 'i' looking for beginning of value")
	})

	t.Run("valid data", func(t *testing.T) {
		t.Parallel()
		r := &fakeRes{}
		err := unmarshal([]byte(validData), r)
		require.NoError(t, err)

		expected := &fakeRes{
			Something: "some data",
		}
		require.Equal(t, r, expected)
	})

	t.Run("bad data format", func(t *testing.T) {
		t.Parallel()
		r := &fakeRes{}
		err := unmarshal([]byte(withBadDataFormat), r)
		require.EqualError(t, err, "failed to decode data into response {\"data\": \"notAndObject\"}: : : : json: cannot unmarshal string into Go value of type clientv2.fakeRes")
	})

	t.Run("bad data format", func(t *testing.T) {
		t.Parallel()
		r := &fakeRes{}
		err := unmarshal([]byte(withBadErrorsFormat), r)
		require.EqualError(t, err, "faild to parse graphql errors. Response content {\"errors\": \"bad\"} - json: cannot unmarshal string into Go struct field GqlErrorList.errors of type gqlerror.List : json: cannot unmarshal string into Go struct field GqlErrorList.errors of type gqlerror.List")
	})
}

func TestParseResponse(t *testing.T) {
	t.Parallel()
	t.Run("single error", func(t *testing.T) {
		t.Parallel()
		r := &fakeRes{}
		err := parseResponse([]byte(qqlSingleErr), 200, r)

		expectedType := &ErrorResponse{}
		require.IsType(t, expectedType, err)

		gqlExpectedType := &gqlerror.List{}
		require.IsType(t, gqlExpectedType, err.(*ErrorResponse).GqlErrors)

		require.Nil(t, err.(*ErrorResponse).NetworkError)
	})

	t.Run("bad error format", func(t *testing.T) {
		t.Parallel()
		r := &fakeRes{}
		err := parseResponse([]byte(withBadErrorsFormat), 200, r)

		expectedType := fmt.Errorf("%w", errors.New("some"))
		require.IsType(t, expectedType, err)
	})

	t.Run("network error with valid gql error response", func(t *testing.T) {
		t.Parallel()
		r := &fakeRes{}
		err := parseResponse([]byte(qqlSingleErr), 400, r)

		expectedType := &ErrorResponse{}
		require.IsType(t, expectedType, err)

		netExpectedType := &HTTPError{}
		require.IsType(t, netExpectedType, err.(*ErrorResponse).NetworkError)

		gqlExpectedType := &gqlerror.List{}
		require.IsType(t, gqlExpectedType, err.(*ErrorResponse).GqlErrors)
	})

	t.Run("network error with not valid gql error response", func(t *testing.T) {
		t.Parallel()
		r := &fakeRes{}
		err := parseResponse([]byte(invalidJSON), 500, r)

		expectedType := &ErrorResponse{}
		require.IsType(t, expectedType, err)

		netExpectedType := &HTTPError{}
		require.IsType(t, netExpectedType, err.(*ErrorResponse).NetworkError)

		require.Nil(t, err.(*ErrorResponse).GqlErrors)
	})

	t.Run("no error", func(t *testing.T) {
		t.Parallel()
		r := &fakeRes{}
		err := parseResponse([]byte(validData), 200, r)

		require.Nil(t, err)
	})
}

func TestChainInterceptor(t *testing.T) {
	t.Parallel()

	someValue := 1
	parentContext := context.WithValue(context.TODO(), "parent", someValue)
	requestMessage := "hoge"
	responseMessage := "foo"
	parentGQLInfo := NewGQLRequestInfo(&Request{
		Query:         "query GQL {id}",
		OperationName: "GQL",
	})
	outputError := fmt.Errorf("some error")
	requireContextValue := func(t *testing.T, ctx context.Context, key string, msg ...interface{}) {
		t.Helper()
		val := ctx.Value(key)
		require.NotNil(t, val, msg...)
		require.Equal(t, someValue, val, msg...)
	}

	req, err := http.NewRequestWithContext(parentContext, http.MethodPost, "https://hogehoge/graphql", bytes.NewBuffer([]byte(requestMessage)))
	require.Nil(t, err)

	first := func(ctx context.Context, req *http.Request, gqlInfo *GQLRequestInfo, res interface{}, next RequestInterceptorFunc) error {
		requireContextValue(t, ctx, "parent", "first must know the parent context value")

		wrappedCtx := context.WithValue(ctx, "first", someValue)

		return next(wrappedCtx, req, gqlInfo, res)
	}

	second := func(ctx context.Context, req *http.Request, gqlInfo *GQLRequestInfo, res interface{}, next RequestInterceptorFunc) error {
		requireContextValue(t, ctx, "parent", "second must know the parent context value")
		requireContextValue(t, ctx, "first", "second must know the first context value")

		wrappedCtx := context.WithValue(ctx, "second", someValue)

		return next(wrappedCtx, req, gqlInfo, res)
	}

	invoker := func(ctx context.Context, req *http.Request, gqlInfo *GQLRequestInfo, res interface{}) error {
		requireContextValue(t, ctx, "parent", "invoker must know the parent context value")
		requireContextValue(t, ctx, "first", "invoker must know the first context value")
		requireContextValue(t, ctx, "second", "invoker must know the second context value")

		return outputError
	}

	chain := ChainInterceptor(first, second)
	err = chain(parentContext, req, parentGQLInfo, responseMessage, invoker)
	require.Equal(t, outputError, err, "chain must return invokers's error")
}

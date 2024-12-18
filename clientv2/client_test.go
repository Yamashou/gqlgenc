package clientv2

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
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
		c := &Client{}
		err := c.unmarshal([]byte(qqlSingleErr), r)
		expectedErr := &GqlErrorList{
			Errors: gqlerror.List{{
				Message: "Field 'nsodes' doesn't exist on type 'RepositoryConnection'",
				Path:    path,
				Locations: []gqlerror.Location{{
					Line:   6,
					Column: 4,
				}},
				Extensions: map[string]any{
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
		c := &Client{}
		err := c.unmarshal([]byte(gqlMultipleErr), r)
		expectedErr := &GqlErrorList{
			Errors: gqlerror.List{
				{
					Message: "Field 'nsodes' doesn't exist on type 'RepositoryConnection'",
					Path:    path1,
					Locations: []gqlerror.Location{{
						Line:   6,
						Column: 4,
					}},
					Extensions: map[string]any{
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
					Extensions: map[string]any{
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
					Extensions: map[string]any{
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
		c := &Client{}
		err := c.unmarshal([]byte(gqlDataAndErr), r)
		expectedErr := &GqlErrorList{
			Errors: gqlerror.List{{
				Message: "Field 'nsodes' doesn't exist on type 'RepositoryConnection'",
				Path:    path,
				Locations: []gqlerror.Location{{
					Line:   6,
					Column: 4,
				}},
				Extensions: map[string]any{
					"code":      "undefinedField",
					"typeName":  "RepositoryConnection",
					"fieldName": "nsodes",
				},
			}},
		}
		require.Equal(t, err, expectedErr)
	})
	t.Run("response data and error still parsed", func(t *testing.T) {
		t.Parallel()
		var path ast.Path
		_ = json.Unmarshal([]byte(`["query GetUser","viewer","repositories","nsodes"]`), &path)
		r := &fakeRes{}
		c := &Client{ParseDataWhenErrors: true}

		err := c.unmarshal([]byte(gqlDataAndErr), r)
		expectedErr := &GqlErrorList{
			Errors: gqlerror.List{{
				Message: "Field 'nsodes' doesn't exist on type 'RepositoryConnection'",
				Path:    path,
				Locations: []gqlerror.Location{{
					Line:   6,
					Column: 4,
				}},
				Extensions: map[string]any{
					"code":      "undefinedField",
					"typeName":  "RepositoryConnection",
					"fieldName": "nsodes",
				},
			}},
		}
		expected := &fakeRes{
			Something: "some data",
		}

		require.Equal(t, err, expectedErr)
		require.Equal(t, r, expected)
	})

	t.Run("invalid json", func(t *testing.T) {
		t.Parallel()
		r := &fakeRes{}
		c := &Client{}
		err := c.unmarshal([]byte(invalidJSON), r)
		require.EqualError(t, err, "failed to decode data invalid: invalid character 'i' looking for beginning of value")
	})

	t.Run("valid data", func(t *testing.T) {
		t.Parallel()
		res := &fakeRes{}
		c := &Client{}
		err := c.unmarshal([]byte(validData), res)
		require.NoError(t, err)

		expected := &fakeRes{
			Something: "some data",
		}
		require.Equal(t, res, expected)
	})

	t.Run("bad data format", func(t *testing.T) {
		t.Parallel()
		r := &fakeRes{}
		c := &Client{}
		err := c.unmarshal([]byte(withBadDataFormat), r)
		require.EqualError(t, err, "failed to decode data into response {\"data\": \"notAndObject\"}: : : : : json: cannot unmarshal string into Go value of type clientv2.fakeRes")
	})

	t.Run("bad data format", func(t *testing.T) {
		t.Parallel()
		r := &fakeRes{}
		c := &Client{}
		err := c.unmarshal([]byte(withBadErrorsFormat), r)
		require.EqualError(t, err, "faild to parse graphql errors. Response content {\"errors\": \"bad\"} - json: cannot unmarshal string into Go struct field GqlErrorList.errors of type gqlerror.List")
	})
}

func TestParseResponse(t *testing.T) {
	t.Parallel()
	t.Run("single error", func(t *testing.T) {
		t.Parallel()
		r := &fakeRes{}
		c := &Client{}
		err := c.parseResponse([]byte(qqlSingleErr), 200, r)

		expectedType := &ErrorResponse{}
		require.IsType(t, expectedType, err)

		gqlExpectedType := &gqlerror.List{}
		require.IsType(t, gqlExpectedType, err.(*ErrorResponse).GqlErrors)

		require.Nil(t, err.(*ErrorResponse).NetworkError)
	})

	t.Run("bad error format", func(t *testing.T) {
		t.Parallel()
		r := &fakeRes{}
		c := &Client{}
		err := c.parseResponse([]byte(withBadErrorsFormat), 200, r)

		expectedType := fmt.Errorf("%w", errors.New("some"))
		require.IsType(t, expectedType, err)
	})

	t.Run("network error with valid gql error response", func(t *testing.T) {
		t.Parallel()
		r := &fakeRes{}
		c := &Client{}
		err := c.parseResponse([]byte(qqlSingleErr), 400, r)

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
		c := &Client{}
		err := c.parseResponse([]byte(invalidJSON), 500, r)

		expectedType := &ErrorResponse{}
		require.IsType(t, expectedType, err)

		netExpectedType := &HTTPError{}
		require.IsType(t, netExpectedType, err.(*ErrorResponse).NetworkError)

		require.Nil(t, err.(*ErrorResponse).GqlErrors)
	})

	t.Run("no error", func(t *testing.T) {
		t.Parallel()
		r := &fakeRes{}
		c := &Client{}
		err := c.parseResponse([]byte(validData), 200, r)

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
	requireContextValue := func(t *testing.T, ctx context.Context, key string, msg ...any) {
		t.Helper()
		val := ctx.Value(key)
		require.NotNil(t, val, msg...)
		require.Equal(t, someValue, val, msg...)
	}

	req, err := http.NewRequestWithContext(parentContext, http.MethodPost, "https://hogehoge/graphql", bytes.NewBufferString(requestMessage))
	require.Nil(t, err)

	first := func(ctx context.Context, req *http.Request, gqlInfo *GQLRequestInfo, res any, next RequestInterceptorFunc) error {
		requireContextValue(t, ctx, "parent", "first must know the parent context value")

		wrappedCtx := context.WithValue(ctx, "first", someValue)

		return next(wrappedCtx, req, gqlInfo, res)
	}

	second := func(ctx context.Context, req *http.Request, gqlInfo *GQLRequestInfo, res any, next RequestInterceptorFunc) error {
		requireContextValue(t, ctx, "parent", "second must know the parent context value")
		requireContextValue(t, ctx, "first", "second must know the first context value")

		wrappedCtx := context.WithValue(ctx, "second", someValue)

		return next(wrappedCtx, req, gqlInfo, res)
	}

	invoker := func(ctx context.Context, req *http.Request, gqlInfo *GQLRequestInfo, res any) error {
		requireContextValue(t, ctx, "parent", "invoker must know the parent context value")
		requireContextValue(t, ctx, "first", "invoker must know the first context value")
		requireContextValue(t, ctx, "second", "invoker must know the second context value")

		return outputError
	}

	chain := ChainInterceptor(first, second)
	err = chain(parentContext, req, parentGQLInfo, responseMessage, invoker)
	require.Equal(t, outputError, err, "chain must return invokers's error")
}

func Test_prepareMultipartFormBody(t *testing.T) {
	t.Parallel()

	t.Run("bad form field", func(t *testing.T) {
		t.Parallel()

		body := new(bytes.Buffer)
		formFields := []FormField{
			{
				Name:  "field",
				Value: make(chan struct{}),
			},
		}

		contentType, err := prepareMultipartFormBody(body, formFields, []MultipartFilesGroup{})

		require.Equal(t, contentType, "")
		require.EqualError(t, err, "encode field: json: unsupported type: chan struct {}")
	})

	t.Run("no errors", func(t *testing.T) {
		t.Parallel()

		body := new(bytes.Buffer)
		formFields := []FormField{
			{
				Name:  "field",
				Value: "value",
			},
		}

		contentType, err := prepareMultipartFormBody(body, formFields, []MultipartFilesGroup{})

		require.Contains(t, contentType, "multipart/form-data; boundary=")
		require.NoError(t, err)
	})
}

func Test_parseMultipartFiles(t *testing.T) {
	t.Parallel()

	t.Run("no files in vars", func(t *testing.T) {
		t.Parallel()

		vars := map[string]any{
			"field":  "val",
			"field2": "val2",
		}

		multipartFilesGroups, mapping, varsMutated := parseMultipartFiles(vars)

		require.Contains(t, varsMutated, "field")
		require.Contains(t, varsMutated, "field2")
		require.Len(t, mapping, 0)
		require.Len(t, multipartFilesGroups, 0)
	})

	t.Run("has file in vars", func(t *testing.T) {
		t.Parallel()

		vars := map[string]any{
			"field": "val",
			"fieldFile": graphql.Upload{
				Filename: "file.txt",
				File:     bytes.NewReader([]byte("content")),
			},
		}

		multipartFilesGroups, mapping, varsMutated := parseMultipartFiles(vars)

		require.Contains(t, varsMutated, "field")
		require.Contains(t, varsMutated, "fieldFile")

		fieldFile, ok := varsMutated["fieldFile"]
		if !ok {
			t.Fatal("fieldFile must present!")
		}

		require.Len(t, mapping, 1)
		require.Len(t, multipartFilesGroups, 1)
		require.Equal(t, multipartFilesGroups[0].IsMultiple, false)
		require.Len(t, multipartFilesGroups[0].Files, 1)
		require.Nil(t, fieldFile)
	})

	t.Run("has optional file in vars", func(t *testing.T) {
		t.Parallel()

		vars := map[string]any{
			"field": "val",
			"fieldFile": &graphql.Upload{
				Filename: "file.txt",
				File:     bytes.NewReader([]byte("content")),
			},
		}

		multipartFilesGroups, mapping, varsMutated := parseMultipartFiles(vars)

		require.Contains(t, varsMutated, "field")
		require.Contains(t, varsMutated, "fieldFile")

		fieldFile, ok := varsMutated["fieldFile"]
		if !ok {
			t.Fatal("fieldFile must present!")
		}

		require.Len(t, mapping, 1)
		require.Len(t, multipartFilesGroups, 1)
		require.Equal(t, multipartFilesGroups[0].IsMultiple, false)
		require.Len(t, multipartFilesGroups[0].Files, 1)
		require.Nil(t, fieldFile)
	})

	t.Run("has no optional file in vars", func(t *testing.T) {
		t.Parallel()

		vars := map[string]any{
			"field":     "val",
			"fieldFile": nil,
		}

		multipartFilesGroups, mapping, varsMutated := parseMultipartFiles(vars)

		require.Contains(t, varsMutated, "field")
		require.Contains(t, varsMutated, "fieldFile")

		fieldFile, ok := varsMutated["fieldFile"]
		if !ok {
			t.Fatal("fieldFile must present!")
		}

		require.Len(t, mapping, 0)
		require.Len(t, multipartFilesGroups, 0)
		require.Nil(t, fieldFile)
	})

	t.Run("has few files in vars", func(t *testing.T) {
		t.Parallel()

		vars := map[string]any{
			"field": "val",
			"fieldFiles": []*graphql.Upload{
				{
					Filename: "file.txt",
					File:     bytes.NewReader([]byte("content")),
				},
				{
					Filename: "file2.txt",
					File:     bytes.NewReader([]byte("content file2")),
				},
			},
		}

		multipartFilesGroups, mapping, varsMutated := parseMultipartFiles(vars)

		require.Contains(t, varsMutated, "field")
		require.Contains(t, varsMutated, "fieldFiles")

		fieldFiles, ok := varsMutated["fieldFiles"]
		if !ok {
			t.Fatal("fieldFile must present!")
		}

		require.Len(t, mapping, 2)
		require.Len(t, multipartFilesGroups, 1)
		require.Equal(t, multipartFilesGroups[0].IsMultiple, true)
		require.Len(t, multipartFilesGroups[0].Files, 2)
		require.ElementsMatch(t, fieldFiles, make([]struct{}, 2))
	})
}

type Number int64

const (
	NumberOne Number = 1
	NumberTwo Number = 2
)

func (n *Number) UnmarshalGQL(v any) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	switch str {
	case "ONE":
		*n = NumberOne
	case "TWO":
		*n = NumberTwo
	default:

		return fmt.Errorf("Number not found Type: %d", n)
	}

	return nil
}

func (n Number) MarshalGQL(w io.Writer) {
	var str string
	switch n {
	case NumberOne:
		str = "ONE"
	case NumberTwo:
		str = "TWO"
	}
	fmt.Fprint(w, strconv.Quote(str))
}

func TestMarshalJSONValueType(t *testing.T) {
	t.Parallel()
	testDate := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	type args struct {
		v any
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "marshal NumberOne",
			args: args{
				v: NumberOne,
			},
			want: []byte(`"ONE"`),
		},
		{
			name: "marshal bool",
			args: args{
				v: true,
			},
			want: []byte("true"),
		},
		{
			name: "marshal int",
			args: args{
				v: 1,
			},
			want: []byte("1"),
		},
		{
			name: "marshal string",
			args: args{
				v: "string",
			},
			want: []byte(`"string"`),
		},
		{
			name: "marshal nil",
			args: args{
				v: nil,
			},
			want: []byte("null"),
		},
		{
			name: "marshal map",
			args: args{
				v: map[Number]string{
					NumberOne: "ONE",
				},
			},
			want: []byte(`{"ONE":"ONE"}`),
		},
		{
			name: "marshal slice",
			args: args{
				v: []Number{NumberOne, NumberTwo},
			},
			want: []byte(`["ONE","TWO"]`),
		},
		{
			name: "marshal time.Time",
			args: args{
				v: testDate,
			},
			want: []byte(`"2021-01-01T00:00:00Z"`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MarshalJSON(tt.args.v)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if !cmp.Equal(tt.want, got) {
				t.Errorf("MarshalJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMarshalJSON(t *testing.T) {
	t.Parallel()
	type Example struct {
		Name   string `json:"name"`
		Number Number `json:"number"`
	}

	var b *Number

	// example nested struct
	type WhereInput struct {
		Not *WhereInput `json:"not,omitempty"`
		ID  *string     `json:"id,omitempty"`
	}

	testID := "1"

	// example with omitted fields
	type Input struct {
		ID   string   `json:"id"`
		Tags []string `json:"tags,omitempty"`
	}

	testDate := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	type args struct {
		v any
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "marshal NumberOne",
			args: args{
				v: map[string]interface{}{"input": NumberOne},
			},
			want: []byte(`{"input":"ONE"}`),
		},
		{
			name: "marshal NumberTwo",
			args: args{
				v: Request{
					OperationName: "query",
					Query:         `query ($input: Number!) { input }`,
					Variables: map[string]any{
						"input": NumberTwo,
					},
				},
			},
			want: []byte(`{"operationName":"query", "query":"query ($input: Number!) { input }","variables":{"input":"TWO"}}`),
		},
		{
			name: "marshal nested",
			args: args{
				v: Request{
					OperationName: "query",
					Query:         `query ($input: Number!) { input }`,
					Variables: map[string]any{
						"where": WhereInput{
							Not: &WhereInput{
								ID: &testID,
							},
						},
					},
				},
			},
			want: []byte(`{"operationName":"query", "query":"query ($input: Number!) { input }","variables":{"where":{"not":{"id":"1"}}}}`),
		},
		{
			name: "marshal nil",
			args: args{
				v: Request{
					OperationName: "query",
					Variables: map[string]any{
						"v": b,
					},
				},
			},
			want: []byte(`{"operationName":"query", "query":"","variables":{"v":null}}`),
		},
		{
			name: "marshal a struct with custom marshaler",
			args: args{
				v: Example{
					Name:   "John",
					Number: NumberOne,
				},
			},
			want: []byte(`{"name":"John","number":"ONE"}`),
		},
		{
			name: "marshal map with custom marshaler",
			args: args{
				v: map[string]any{
					"number": NumberOne,
					"example2": &Example{
						Name:   "John",
						Number: NumberOne,
					},
				},
			},
			want: []byte(`{"example2":{"name":"John","number":"ONE"},"number":"ONE"}`),
		},
		{
			name: "marshal time.Time",
			args: args{
				v: struct {
					Time *time.Time `json:"time"`
				}{
					Time: &testDate,
				},
			},
			want: []byte(`{"time":"2021-01-01T00:00:00Z"}`),
		},
		{
			name: "marshal omitted fields",
			args: args{
				v: Request{
					OperationName: "query",
					Query:         `query ($input: Number!) { input }`,
					Variables: map[string]any{
						"input": Input{
							ID: "1",
						},
					},
				},
			},
			want: []byte(`{"operationName":"query", "query":"query ($input: Number!) { input }","variables":{"input":{"id":"1"}}}`),
		},
		{
			name: "marshal fields",
			args: args{
				v: Request{
					OperationName: "query",
					Query:         `query ($input: Number!) { input }`,
					Variables: map[string]any{
						"input": Input{
							ID:   "1",
							Tags: []string{"tag1", "tag2"},
						},
					},
				},
			},
			want: []byte(`{"operationName":"query", "query":"query ($input: Number!) { input }","variables":{"input":{"id":"1", "tags":["tag1","tag2"]}}}`),
		},
		{
			name: "marshal time.Time",
			args: args{
				v: struct {
					T struct {
						Time time.Time `json:"time"`
					}
				}{
					T: struct {
						Time time.Time `json:"time"`
					}{
						Time: testDate,
					},
				},
			},
			want: []byte(`{"T":{"time":"2021-01-01T00:00:00Z"}}`),
		},
		{
			name: "marshal uuid",
			args: args{
				v: struct {
					T struct {
						UUID uuid.UUID `json:"uuid"`
					}
				}{
					T: struct {
						UUID uuid.UUID `json:"uuid"`
					}{
						UUID: uuid.MustParse("0bd42821-463a-4224-a41b-c5861fc91268"),
					},
				},
			},
			want: []byte(`{"T":{"uuid":"0bd42821-463a-4224-a41b-c5861fc91268"}}`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MarshalJSON(tt.args.v)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			var gotMap, wantMap map[string]any
			if err := json.Unmarshal(got, &gotMap); err != nil {
				t.Errorf("Failed to unmarshal 'got': %s", string(got))
				return
			}
			if err := json.Unmarshal(tt.want, &wantMap); err != nil {
				t.Errorf("Failed to unmarshal err: %s", err)
				return
			}

			if !cmp.Equal(gotMap, wantMap) {
				t.Errorf("MarshalJSON() got = %v, want %v", gotMap, wantMap)
			}
		})
	}
}

func TestUnsafeChainInterceptor(t *testing.T) {
	t.Run("should modify values through interceptors", func(t *testing.T) {
		// Prepare test values
		originalCtx := context.Background()
		originalReq, _ := http.NewRequest("POST", "http://example.com", nil)
		originalGqlInfo := &GQLRequestInfo{
			Request: &Request{Query: "original"},
		}
		originalRes := "original"

		// First interceptor: Add value to context
		interceptor1 := func(ctx context.Context, req *http.Request, gqlInfo *GQLRequestInfo, res any, next RequestInterceptorFunc) error {
			ctx = context.WithValue(ctx, "key1", "value1")
			return next(ctx, req, gqlInfo, res)
		}

		// Second interceptor: Modify request header
		interceptor2 := func(ctx context.Context, req *http.Request, gqlInfo *GQLRequestInfo, res any, next RequestInterceptorFunc) error {
			req.Header.Set("X-Test", "test-value")
			return next(ctx, req, gqlInfo, res)
		}

		// Third interceptor: Modify GQLInfo and response
		interceptor3 := func(ctx context.Context, req *http.Request, gqlInfo *GQLRequestInfo, res any, next RequestInterceptorFunc) error {
			gqlInfo.Request.Query = "modified"
			return next(ctx, req, gqlInfo, "modified")
		}

		// Final handler: Verify modified values
		finalHandler := func(ctx context.Context, req *http.Request, gqlInfo *GQLRequestInfo, res any) error {
			// Verify context
			if v := ctx.Value("key1"); v != "value1" {
				t.Errorf("context value not propagated, got %v", v)
			}

			// Verify request header
			if v := req.Header.Get("X-Test"); v != "test-value" {
				t.Errorf("request header not modified, got %v", v)
			}

			// Verify GQLInfo
			if gqlInfo.Request.Query != "modified" {
				t.Errorf("GQLInfo not modified, got %v", gqlInfo.Request.Query)
			}

			// Verify response
			if res != "modified" {
				t.Errorf("response not modified, got %v", res)
			}

			return nil
		}

		// Create interceptor chain
		chain := UnsafeChainInterceptor(interceptor1, interceptor2, interceptor3)

		// Execute chain
		err := chain(originalCtx, originalReq, originalGqlInfo, originalRes, finalHandler)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("should properly propagate errors", func(t *testing.T) {
		expectedError := errors.New("test error")
		
		// Interceptor that returns an error
		errorInterceptor := func(ctx context.Context, req *http.Request, gqlInfo *GQLRequestInfo, res any, next RequestInterceptorFunc) error {
			return expectedError
		}

		// Create chain
		chain := UnsafeChainInterceptor(errorInterceptor)

		// Execute chain
		err := chain(
			context.Background(),
			&http.Request{},
			&GQLRequestInfo{},
			nil,
			func(ctx context.Context, req *http.Request, gqlInfo *GQLRequestInfo, res any) error {
				return nil
			},
		)

		if err != expectedError {
			t.Errorf("expected error %v, got %v", expectedError, err)
		}
	})

	t.Run("should execute interceptors in correct order", func(t *testing.T) {
		var order []int
		
		// Create interceptors that record execution order
		makeInterceptor := func(id int) RequestInterceptor {
			return func(ctx context.Context, req *http.Request, gqlInfo *GQLRequestInfo, res any, next RequestInterceptorFunc) error {
				order = append(order, id)
				err := next(ctx, req, gqlInfo, res)
				order = append(order, -id) // Record return order as well
				return err
			}
		}

		// Create chain
		chain := UnsafeChainInterceptor(makeInterceptor(1), makeInterceptor(2), makeInterceptor(3))

		// Execute chain
		_ = chain(
			context.Background(),
			&http.Request{},
			&GQLRequestInfo{},
			nil,
			func(ctx context.Context, req *http.Request, gqlInfo *GQLRequestInfo, res any) error {
				order = append(order, 0) // Record execution of final handler
				return nil
			},
		)

		// Expected execution order: 1 -> 2 -> 3 -> 0 -> -3 -> -2 -> -1
		expected := []int{1, 2, 3, 0, -3, -2, -1}
		if !reflect.DeepEqual(order, expected) {
			t.Errorf("unexpected execution order\nexpected: %v\ngot: %v", expected, order)
		}
	})
}

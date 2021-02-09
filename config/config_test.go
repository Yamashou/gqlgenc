package config

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"

	"github.com/99designs/gqlgen/codegen/config"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	t.Parallel()
	t.Run("config does not exist", func(t *testing.T) {
		t.Parallel()
		_, err := LoadConfig("doesnotexist.yml")
		require.Error(t, err)
	})

	t.Run("malformed config", func(t *testing.T) {
		t.Parallel()
		_, err := LoadConfig("testdata/cfg/malformedconfig.yml")
		require.EqualError(t, err, "unable to parse config: yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str `asdf` into config.Config")
	})

	t.Run("'schema' and 'endpoint' both specified", func(t *testing.T) {
		t.Parallel()
		_, err := LoadConfig("testdata/cfg/schema_endpoint.yml")
		require.EqualError(t, err, "'schema' and 'endpoint' both specified. Use schema to load from a local file, use endpoint to load from a remote server (using introspection)")
	})

	t.Run("neither 'schema' nor 'endpoint' specified", func(t *testing.T) {
		t.Parallel()
		_, err := LoadConfig("testdata/cfg/no_source.yml")
		require.EqualError(t, err, "neither 'schema' nor 'endpoint' specified. Use schema to load from a local file, use endpoint to load from a remote server (using introspection)")
	})

	t.Run("unknown keys", func(t *testing.T) {
		t.Parallel()
		_, err := LoadConfig("testdata/cfg/unknownkeys.yml")
		require.EqualError(t, err, "unable to parse config: yaml: unmarshal errors:\n  line 3: field unknown not found in type config.Config")
	})

	t.Run("globbed filenames", func(t *testing.T) {
		t.Parallel()
		c, err := LoadConfig("testdata/cfg/glob.yml")
		require.NoError(t, err)

		if runtime.GOOS == "windows" {
			require.Equal(t, c.SchemaFilename[0], `testdata\cfg\glob\bar\bar with spaces.graphql`)
			require.Equal(t, c.SchemaFilename[1], `testdata\cfg\glob\foo\foo.graphql`)
		} else {
			require.Equal(t, c.SchemaFilename[0], "testdata/cfg/glob/bar/bar with spaces.graphql")
			require.Equal(t, c.SchemaFilename[1], "testdata/cfg/glob/foo/foo.graphql")
		}
	})

	t.Run("unwalkable path", func(t *testing.T) {
		t.Parallel()
		_, err := LoadConfig("testdata/cfg/unwalkable.yml")
		if runtime.GOOS == "windows" {
			require.EqualError(t, err, "failed to walk schema at root not_walkable/: CreateFile not_walkable/: The system cannot find the file specified.")
		} else {
			require.EqualError(t, err, "failed to walk schema at root not_walkable/: lstat not_walkable/: no such file or directory")
		}
	})

	t.Run("generate", func(t *testing.T) {
		t.Parallel()
		c, err := LoadConfig("testdata/cfg/generate.yml")
		require.NoError(t, err)
		require.Equal(t, c.Generate.Suffix.Mutation, "Bar")
		require.Equal(t, c.Generate.Suffix.Query, "Foo")
		require.Equal(t, c.Generate.Prefix.Mutation, "Hoge")
		require.Equal(t, c.Generate.Prefix.Query, "Data")
	})
}

func TestLoadConfig_LoadSchema(t *testing.T) {
	t.Parallel()

	t.Run("correct schema", func(t *testing.T) {
		t.Parallel()

		mockServer, closeServer := newMockRemoteServer(t, responseFromFile("testdata/remote/response_ok.json"))
		defer closeServer()

		config := &Config{
			GQLConfig: &config.Config{},
			Endpoint: &EndPointConfig{
				URL: mockServer.URL,
			},
		}

		err := config.LoadSchema(context.Background())
		require.NoError(t, err)
	})

	t.Run("invalid schema", func(t *testing.T) {
		t.Parallel()

		mockServer, closeServer := newMockRemoteServer(t, responseFromFile("testdata/remote/response_invalid_schema.json"))
		defer closeServer()

		config := &Config{
			GQLConfig: &config.Config{},
			Endpoint: &EndPointConfig{
				URL: mockServer.URL,
			},
		}

		err := config.LoadSchema(context.Background())
		require.Equal(t, "load remote schema failed: validation error: remote:0: OBJECT must define one or more fields.", err.Error())
	})
}

type mockRemoteServer struct {
	*httptest.Server
	body []byte
}

func newMockRemoteServer(t *testing.T, response interface{}) (mock *mockRemoteServer, closeServer func()) {
	t.Helper()

	mock = &mockRemoteServer{
		Server: httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			var err error
			mock.body, err = ioutil.ReadAll(req.Body)
			require.NoError(t, err)

			var responseBody []byte
			switch v := response.(type) {
			case json.RawMessage:
				responseBody = v
			case responseFromFile:
				responseBody = v.load(t)
			default:
				responseBody, err = json.Marshal(response)
				require.NoError(t, err)
			}

			_, err = rw.Write(responseBody)
			require.NoError(t, err)
		})),
	}

	return mock, func() { mock.Close() }
}

type responseFromFile string

func (f responseFromFile) load(t *testing.T) []byte {
	t.Helper()

	content, err := ioutil.ReadFile(string(f))
	require.NoError(t, err)

	return content
}

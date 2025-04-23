package config

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"testing"

	"github.com/99designs/gqlgen/codegen/config"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func ptr[T any](t T) *T {
	return &t
}

func TestLoadConfig(t *testing.T) {
	t.Parallel()
	type want struct {
		err        bool
		errMessage string
		config     *Config
	}

	tests := []struct {
		name string
		file string
		want want
	}{
		{
			name: "config does not exist",
			file: "doesnotexist.yml",
			want: want{
				err: true,
			},
		},
		{
			name: "malformed config",
			file: "testdata/cfg/malformedconfig.yml",
			want: want{
				err:        true,
				errMessage: "unable to parse config: [1:1] string was used where mapping is expected\n>  1 | asdf\n       ^\n",
			},
		},
		{
			name: "'schema' and 'endpoint' both specified",
			file: "testdata/cfg/schema_endpoint.yml",
			want: want{
				err:        true,
				errMessage: "'schema' and 'endpoint' both specified. Use schema to load from a local file, use endpoint to load from a remote server (using introspection)",
			},
		},
		{
			name: "neither 'schema' nor 'endpoint' specified",
			file: "testdata/cfg/no_source.yml",
			want: want{
				err:        true,
				errMessage: "neither 'schema' nor 'endpoint' specified. Use schema to load from a local file, use endpoint to load from a remote server (using introspection)",
			},
		},
		{
			name: "unknown keys",
			file: "testdata/cfg/unknownkeys.yml",
			want: want{
				err:        true,
				errMessage: "unknown field \"unknown\"",
			},
		},
		{
			name: "nullable input omittable",
			file: "testdata/cfg/nullable_input_omittable.yml",
			want: want{
				config: &Config{
					GQLGencConfig: &GQLGencConfig{
						Query: []string{"./queries/*.graphql"},
						QueryGen: config.PackageConfig{
							Package: "gen",
						},
						ClientGen: config.PackageConfig{
							Package: "gen",
						},
					},
					GQLGenConfig: &config.Config{
						SchemaFilename: config.StringList{
							"testdata/cfg/glob/bar/bar with spaces.graphql",
							"testdata/cfg/glob/foo/foo.graphql",
						},
						Exec: config.ExecConfig{
							Filename: "generated.go",
						},
						Model: config.PackageConfig{
							Filename: "./gen/models_gen.go",
						},
						Federation: config.PackageConfig{
							Filename: "generated.go",
						},
						Resolver: config.ResolverConfig{
							Filename: "generated.go",
						},
						NullableInputOmittable: true,
						Directives:             map[string]config.DirectiveConfig{},
						GoInitialisms:          config.GoInitialismsConfig{},
					},
				},
			},
		},
		{
			name: "omitempty, omitzero",
			file: "testdata/cfg/omitempty_omitzero.yml",
			want: want{
				config: &Config{
					GQLGencConfig: &GQLGencConfig{
						Query: []string{"./queries/*.graphql"},
						QueryGen: config.PackageConfig{
							Package: "gen",
						},
						ClientGen: config.PackageConfig{
							Package: "gen",
						},
					},
					GQLGenConfig: &config.Config{
						SchemaFilename: config.StringList{
							"testdata/cfg/glob/bar/bar with spaces.graphql",
							"testdata/cfg/glob/foo/foo.graphql",
						},
						Exec: config.ExecConfig{
							Filename: "generated.go",
						},
						Model: config.PackageConfig{
							Filename: "./gen/models_gen.go",
						},
						Federation: config.PackageConfig{
							Filename: "generated.go",
						},
						Resolver: config.ResolverConfig{
							Filename: "generated.go",
						},
						EnableModelJsonOmitemptyTag: ptr(true),
						EnableModelJsonOmitzeroTag:  ptr(true),
						Directives:                  map[string]config.DirectiveConfig{},
						GoInitialisms:               config.GoInitialismsConfig{},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg, err := LoadConfig(tt.file)

			if tt.want.err {
				if err == nil {
					t.Errorf("LoadConfig() error = nil, want error")
					return
				}
				if tt.want.errMessage != "" && !containsString(err.Error(), tt.want.errMessage) {
					t.Errorf("LoadConfig() error = %v, want error containing %v", err, tt.want.errMessage)
				}
				return
			}

			if err != nil {
				t.Errorf("LoadConfig() error = %v, want nil", err)
				return
			}

			if tt.want.config != nil {
				opts := []cmp.Option{
					cmpopts.IgnoreFields(config.Config{}, "Sources"),
					cmpopts.IgnoreFields(config.PackageConfig{}, "Filename"),
				}
				if diff := cmp.Diff(tt.want.config, cfg, opts...); diff != "" {
					t.Errorf("LoadConfig() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestLoadConfigWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific test on non-Windows platform")
	}

	t.Parallel()

	// Glob filenames test for Windows
	t.Run("globbed filenames on Windows", func(t *testing.T) {
		t.Parallel()
		cfg, err := LoadConfig("testdata/cfg/glob.yml")
		if err != nil {
			t.Errorf("LoadConfig() error = %v, want nil", err)
			return
		}
		want := `testdata\cfg\glob\bar\bar with spaces.graphql`
		if got := cfg.GQLGenConfig.SchemaFilename[0]; got != want {
			t.Errorf("LoadConfig() SchemaFilename[0] = %v, want %v", got, want)
		}
		want = `testdata\cfg\glob\foo\foo.graphql`
		if got := cfg.GQLGenConfig.SchemaFilename[1]; got != want {
			t.Errorf("LoadConfig() SchemaFilename[1] = %v, want %v", got, want)
		}
	})

	// Unwalkable path test for Windows
	t.Run("unwalkable path on Windows", func(t *testing.T) {
		t.Parallel()
		_, err := LoadConfig("testdata/cfg/unwalkable.yml")
		want := "failed to walk schema at root not_walkable/: CreateFile not_walkable/: The system cannot find the file specified."
		if err == nil || err.Error() != want {
			t.Errorf("LoadConfig() error = %v, want %v", err, want)
		}
	})
}

func TestLoadConfigNonWindows(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping non-Windows test on Windows platform")
	}

	t.Parallel()

	// Glob filenames test for non-Windows
	t.Run("globbed filenames on non-Windows", func(t *testing.T) {
		t.Parallel()
		cfg, err := LoadConfig("testdata/cfg/glob.yml")
		if err != nil {
			t.Errorf("LoadConfig() error = %v, want nil", err)
			return
		}
		want := "testdata/cfg/glob/bar/bar with spaces.graphql"
		if got := cfg.GQLGenConfig.SchemaFilename[0]; got != want {
			t.Errorf("LoadConfig() SchemaFilename[0] = %v, want %v", got, want)
		}
		want = "testdata/cfg/glob/foo/foo.graphql"
		if got := cfg.GQLGenConfig.SchemaFilename[1]; got != want {
			t.Errorf("LoadConfig() SchemaFilename[1] = %v, want %v", got, want)
		}
	})

	// Unwalkable path test for non-Windows
	t.Run("unwalkable path on non-Windows", func(t *testing.T) {
		t.Parallel()
		_, err := LoadConfig("testdata/cfg/unwalkable.yml")
		want := "failed to walk schema at root not_walkable/: lstat not_walkable/: no such file or directory"
		if err == nil || err.Error() != want {
			t.Errorf("LoadConfig() error = %v, want %v", err, want)
		}
	})
}

func TestLoadConfig_LoadSchema(t *testing.T) {
	t.Parallel()
	type want struct {
		err        bool
		errMessage string
		config     *Config
	}

	tests := []struct {
		name         string
		responseFile string
		want         want
	}{
		// TODO: LoadLocalSchema
		{
			name:         "correct remote schema",
			responseFile: "testdata/remote/response_ok.json",
			want: want{
				config: &Config{
					GQLGencConfig: &GQLGencConfig{
						Endpoint: &EndPointConfig{},
					},
					GQLGenConfig: &config.Config{},
				},
			},
		},
		{
			name:         "invalid remote schema",
			responseFile: "testdata/remote/response_invalid_schema.json",
			want: want{
				err:        true,
				errMessage: "OBJECT Query: must define one or more fields",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockServer, closeServer := newMockRemoteServer(t, responseFromFile(tt.responseFile))
			defer closeServer()

			cfg := &Config{
				GQLGenConfig: &config.Config{},
				GQLGencConfig: &GQLGencConfig{
					Endpoint: &EndPointConfig{
						URL: mockServer.URL,
					},
				},
			}

			err := cfg.LoadSchema(context.Background())
			if tt.want.err {
				if err == nil {
					t.Errorf("LoadSchema() error = nil, want error")
					return
				}
				if tt.want.errMessage != "" && !containsString(err.Error(), tt.want.errMessage) {
					t.Errorf("LoadSchema() error = %v, want error containing %v", err, tt.want.errMessage)
				}
				return
			}

			if err != nil {
				t.Errorf("LoadSchema() error = %v, want nil", err)
				return
			}

			if tt.want.config != nil {
				opts := []cmp.Option{
					cmpopts.IgnoreFields(config.Config{}, "Schema"),
					cmpopts.IgnoreFields(EndPointConfig{}, "URL"),
				}
				if diff := cmp.Diff(tt.want.config, cfg, opts...); diff != "" {
					t.Errorf("LoadSchema() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

// containsString は文字列sがsubstringを含むかどうかを確認します
func containsString(s, substring string) bool {
	if len(s) < len(substring) || substring == "" {
		return false
	}
	for i := 0; i <= len(s)-len(substring); i++ {
		if s[i:i+len(substring)] == substring {
			return true
		}
	}
	return false
}

type mockRemoteServer struct {
	*httptest.Server
	body []byte
}

func newMockRemoteServer(t *testing.T, response any) (mock *mockRemoteServer, closeServer func()) {
	t.Helper()

	mock = &mockRemoteServer{
		Server: httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
			var err error
			mock.body, err = io.ReadAll(req.Body)
			if err != nil {
				t.Errorf("Failed to read request body: %v", err)
			}

			var responseBody []byte
			switch v := response.(type) {
			case json.RawMessage:
				responseBody = v
			case responseFromFile:
				responseBody = v.load(t)
			default:
				responseBody, err = json.Marshal(response)
				if err != nil {
					t.Errorf("Failed to marshal response: %v", err)
				}
			}

			_, err = writer.Write(responseBody)
			if err != nil {
				t.Errorf("Failed to write response: %v", err)
			}
		})),
	}

	return mock, func() { mock.Close() }
}

type responseFromFile string

func (f responseFromFile) load(t *testing.T) []byte {
	t.Helper()

	content, err := os.ReadFile(string(f))
	if err != nil {
		t.Errorf("Failed to read file %s: %v", string(f), err)
	}

	return content
}

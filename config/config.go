package config

import (
	"context"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/99designs/gqlgen/codegen/config"
	"github.com/Yamashou/gqlgenc/client"
	"github.com/Yamashou/gqlgenc/introspection"
	"github.com/pkg/errors"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/validator"
	"golang.org/x/xerrors"
	"gopkg.in/yaml.v2"
)

// Config extends the gqlgen basic config
// and represents the config file
type Config struct {
	SchemaFilename []string             `yaml:"schema,omitempty"`
	Model          config.PackageConfig `yaml:"model,omitempty"`
	Client         config.PackageConfig `yaml:"client,omitempty"`
	Models         config.TypeMap       `yaml:"models,omitempty"`
	Endpoint       *EndPointConfig      `yaml:"endpoint,omitempty"`
	Query          []string             `yaml:"query"`

	// gqlgen config struct
	GQLConfig *config.Config `yaml:"-"`
}

// EndPointConfig are the allowed options for the 'endpoint' config
type EndPointConfig struct {
	URL     string            `yaml:"url"`
	Headers map[string]string `yaml:"headers,omitempty"`
}

func findCfg(fileName string) (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", xerrors.Errorf("unable to get working dir to findCfg: %w", err)
	}

	cfg := findCfgInDir(dir, fileName)

	if cfg == "" {
		return "", os.ErrNotExist
	}

	return cfg, nil
}

func findCfgInDir(dir, fileName string) string {
	path := filepath.Join(dir, fileName)

	return path
}

// LoadConfig loads and parses the config gqlgenc config
func LoadConfig(filename string) (*Config, error) {
	var cfg Config
	file, err := findCfg(filename)
	if err != nil {
		return nil, xerrors.Errorf("unable to get file path: %w", err)
	}
	b, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, xerrors.Errorf("unable to read config: %w", err)
	}

	confContent := []byte(os.ExpandEnv(string(b)))
	if err := yaml.UnmarshalStrict(confContent, &cfg); err != nil {
		return nil, xerrors.Errorf("unable to parse config: %w", err)
	}
	if cfg.SchemaFilename != nil && cfg.Endpoint != nil {
		return nil, xerrors.Errorf("'schema' and 'endpoint' both specified. Use schema to load from a local file, use endpoint to load from a remote server (using introspection)")
	} else if cfg.SchemaFilename == nil && cfg.Endpoint == nil {
		return nil, xerrors.Errorf("neither 'schema' nor 'endpoint' specified. Use schema to load from a local file, use endpoint to load from a remote server (using introspection)")
	}

	models := make(config.TypeMap)
	if cfg.Models != nil {
		models = cfg.Models
	}

	sources := []*ast.Source{}

	for _, filename := range cfg.SchemaFilename {
		filename = filepath.ToSlash(filename)
		var err error
		var schemaRaw []byte
		schemaRaw, err = ioutil.ReadFile(filename)
		if err != nil {
			return nil, errors.Wrap(err, "unable to open schema")
		}

		sources = append(sources, &ast.Source{Name: filename, Input: string(schemaRaw)})
	}

	cfg.GQLConfig = &config.Config{
		Model:  cfg.Model,
		Models: models,
		// TODO: gqlgen must be set exec but client not used
		Exec:       config.PackageConfig{Filename: "generated.go"},
		Directives: map[string]config.DirectiveConfig{},
		Sources:    sources,
	}

	if err := cfg.Client.Check(); err != nil {
		return nil, xerrors.Errorf("config.exec: %w", err)
	}

	return &cfg, nil
}

// LoadSchema load and parses the schema from a local file or a remote server
func (c *Config) LoadSchema(ctx context.Context) error {
	var schema *ast.Schema

	if c.SchemaFilename != nil {
		s, err := c.loadLocalSchema(ctx)
		if err != nil {
			return xerrors.Errorf("load local schema failed: %w", err)
		}
		schema = s
	} else {
		s, err := c.loadRemoteSchema(ctx)
		if err != nil {
			return xerrors.Errorf("load remote schema failed: %w", err)
		}
		schema = s
	}

	if schema.Query == nil {
		schema.Query = &ast.Definition{
			Kind: ast.Object,
			Name: "Query",
		}
		schema.Types["Query"] = schema.Query
	}

	c.GQLConfig.Schema = schema

	return nil
}

func (c *Config) loadRemoteSchema(ctx context.Context) (*ast.Schema, error) {
	addHeader := func(req *http.Request) {
		for key, value := range c.Endpoint.Headers {
			req.Header.Set(key, value)
		}
	}
	gqlclient := client.NewClient(http.DefaultClient, c.Endpoint.URL, addHeader)

	var res introspection.Query
	if err := gqlclient.Post(ctx, introspection.Introspection, &res, nil); err != nil {
		return nil, xerrors.Errorf("introspection query failed: %w", err)
	}

	schema, err := validator.ValidateSchemaDocument(introspection.ParseIntrospectionQuery(res))
	if err != nil {
		return nil, xerrors.Errorf("validation error: %w", err)
	}

	return schema, nil
}

func (c *Config) loadLocalSchema(ctx context.Context) (*ast.Schema, error) {
	schema, err := gqlparser.LoadSchema(c.GQLConfig.Sources...)
	if err != nil {
		return nil, err
	}
	return schema, nil
}

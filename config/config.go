package config

import (
	"bytes"
	"context"
	"fmt"
	gqlgenconfig "github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/plugin/federation"
	"github.com/Yamashou/gqlgenc/v3/client"
	"github.com/Yamashou/gqlgenc/v3/introspection"
	"github.com/goccy/go-yaml"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/validator"
	"net/http"
	"os"
	"slices"
	"strings"
)

// Config extends the gqlgen basic config
// and represents the config file
type Config struct {
	GQLGencConfig *GQLGencConfig       `yaml:"gqlgenc"`
	GQLGenConfig  *gqlgenconfig.Config `yaml:"gqlgen"`
}

type GQLGencConfig struct {
	Query          []string                   `yaml:"query"`
	QueryGen       gqlgenconfig.PackageConfig `yaml:"querygen,omitempty"`
	ClientGen      gqlgenconfig.PackageConfig `yaml:"clientgen,omitempty"`
	Endpoint       *EndPointConfig            `yaml:"endpoint,omitempty"`
	UsedOnlyModels *bool                      `yaml:"usedModelsOnly,omitempty"`
}

// EndPointConfig are the allowed options for the 'endpoint' config
type EndPointConfig struct {
	URL     string            `yaml:"url"`
	Headers map[string]string `yaml:"headers,omitempty"`
}

// Load loads and parses the config gqlgenc config
func Load(configFilename string) (*Config, error) {
	configContent, err := os.ReadFile(configFilename)
	if err != nil {
		return nil, fmt.Errorf("unable to read config: %w", err)
	}
	var cfg Config
	decoder := yaml.NewDecoder(bytes.NewReader([]byte(os.ExpandEnv(string(configContent)))), yaml.DisallowUnknownField())
	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("unable to parse config: %w", err)
	}

	///////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	// gqlgen

	// validation
	if cfg.GQLGenConfig.SchemaFilename != nil && cfg.GQLGencConfig.Endpoint != nil {
		return nil, fmt.Errorf("'schema' and 'endpoint' both specified. Use schema to load from a local file, use endpoint to load from a remote server (using introspection)")
	}

	if cfg.GQLGenConfig.SchemaFilename == nil && cfg.GQLGencConfig.Endpoint == nil {
		return nil, fmt.Errorf("neither 'schema' nor 'endpoint' specified. Use schema to load from a local file, use endpoint to load from a remote server (using introspection)")
	}

	if err := cfg.GQLGenConfig.Model.Check(); err != nil {
		return nil, fmt.Errorf("model: %w", err)
	}

	// Fill gqlgen config fields
	// https://github.com/99designs/gqlgen/blob/3a31a752df764738b1f6e99408df3b169d514784/codegen/config/config.go#L120
	schemaFilename, err := schemaFilenames(cfg.GQLGenConfig.SchemaFilename)
	if err != nil {
		return nil, err
	}
	cfg.GQLGenConfig.SchemaFilename = schemaFilename

	sources, err := schemaFileSources(cfg.GQLGenConfig.SchemaFilename)
	if err != nil {
		return nil, err
	}
	cfg.GQLGenConfig.Sources = sources

	if cfg.GQLGenConfig.Federation.Version != 0 {
		fedPlugin, err := federation.New(cfg.GQLGenConfig.Federation.Version, cfg.GQLGenConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create federation plugin: %w", err)
		}
		if sources, err := fedPlugin.InjectSourcesEarly(); err == nil {
			cfg.GQLGenConfig.Sources = append(cfg.GQLGenConfig.Sources, sources...)
		} else {
			return nil, fmt.Errorf("failed to inject federation directives: %w", err)
		}
	}

	// gqlgen must be followings parameters
	cfg.GQLGenConfig.Directives = make(map[string]gqlgenconfig.DirectiveConfig)
	cfg.GQLGenConfig.Exec = gqlgenconfig.ExecConfig{Filename: "generated.go"}
	cfg.GQLGenConfig.Resolver = gqlgenconfig.ResolverConfig{Filename: "generated.go"}
	cfg.GQLGenConfig.Federation = gqlgenconfig.PackageConfig{Filename: "generated.go"}

	///////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	// gqlgenc

	// validation
	if err := cfg.GQLGencConfig.QueryGen.Check(); err != nil {
		return nil, fmt.Errorf("querygen: %w", err)
	}

	if err := cfg.GQLGencConfig.ClientGen.Check(); err != nil {
		return nil, fmt.Errorf("clientgen: %w", err)
	}

	return &cfg, nil
}

func (c *Config) Init(ctx context.Context) error {
	if err := c.loadSchema(ctx); err != nil {
		return fmt.Errorf("failed to load schema: %w", err)
	}

	if err := c.GQLGenConfig.Init(); err != nil {
		return fmt.Errorf("generating core failed: %w", err)
	}

	// sort Implements to ensure a deterministic output
	for _, implements := range c.GQLGenConfig.Schema.Implements {
		slices.SortFunc(implements, func(a, b *ast.Definition) int {
			return strings.Compare(a.Name, b.Name)
		})
	}

	return nil
}

// loadSchema load and parses the schema from a local file or a remote server
func (c *Config) loadSchema(ctx context.Context) error {
	// TODO: SchemaFilenameをconfigに指定しなかった場合のtest
	if c.GQLGenConfig.SchemaFilename != nil {
		err := c.GQLGenConfig.LoadSchema()
		if err != nil {
			return fmt.Errorf("load local schema failed: %w", err)
		}
	} else {
		err := c.loadRemoteSchema(ctx)
		if err != nil {
			return fmt.Errorf("load remote schema failed: %w", err)
		}
	}

	return nil
}

func (c *Config) loadRemoteSchema(ctx context.Context) error {
	header := make(http.Header, len(c.GQLGencConfig.Endpoint.Headers))
	for key, value := range c.GQLGencConfig.Endpoint.Headers {
		header[key] = []string{value}
	}
	transport := TransportAppend(
		http.DefaultTransport,
		NewHeaderTransport(func(ctx context.Context) http.Header { return header }),
	)
	httpClient := &http.Client{Transport: transport}
	gqlgencClient := client.NewClient(c.GQLGencConfig.Endpoint.URL, client.WithHTTPClient(httpClient))

	var res introspection.Query
	if err := gqlgencClient.Post(ctx, "Query", introspection.Introspection, nil, &res); err != nil {
		return fmt.Errorf("introspection query failed: %w", err)
	}

	schema, err := validator.ValidateSchemaDocument(introspection.ParseIntrospectionQuery(c.GQLGencConfig.Endpoint.URL, res))
	if err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	if schema.Query == nil {
		schema.Query = &ast.Definition{
			Kind: ast.Object,
			Name: "Query",
		}
		schema.Types["Query"] = schema.Query
	}

	c.GQLGenConfig.Schema = schema

	return nil
}

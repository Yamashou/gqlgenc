package config

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	t.Run("config does not exist", func(t *testing.T) {
		_, err := LoadConfig("doesnotexist.yml")
		require.Error(t, err)
	})

	t.Run("malformed config", func(t *testing.T) {
		_, err := LoadConfig("testdata/cfg/malformedconfig.yml")
		require.EqualError(t, err, "unable to parse config: yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str `asdf` into config.Config")
	})

	t.Run("'schema' and 'endpoint' both specified", func(t *testing.T) {
		_, err := LoadConfig("testdata/cfg/schema_endpoint.yml")
		require.EqualError(t, err, "'schema' and 'endpoint' both specified. Use schema to load from a local file, use endpoint to load from a remote server (using introspection)")
	})

	t.Run("neither 'schema' nor 'endpoint' specified", func(t *testing.T) {
		_, err := LoadConfig("testdata/cfg/no_source.yml")
		require.EqualError(t, err, "neither 'schema' nor 'endpoint' specified. Use schema to load from a local file, use endpoint to load from a remote server (using introspection)")
	})

	t.Run("unknown keys", func(t *testing.T) {
		_, err := LoadConfig("testdata/cfg/unknownkeys.yml")
		require.EqualError(t, err, "unable to parse config: yaml: unmarshal errors:\n  line 3: field unknown not found in type config.Config")
	})

	t.Run("globbed filenames", func(t *testing.T) {
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
		_, err := LoadConfig("testdata/cfg/unwalkable.yml")
		if runtime.GOOS == "windows" {
			require.EqualError(t, err, "failed to walk schema at root not_walkable/: CreateFile not_walkable/: The system cannot find the file specified.")
		} else {
			require.EqualError(t, err, "failed to walk schema at root not_walkable/: lstat not_walkable/: no such file or directory")
		}
	})
}

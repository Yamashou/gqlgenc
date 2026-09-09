package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveVersion(t *testing.T) {
	t.Parallel()

	t.Run("build override wins", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, "v1.2.3", resolveVersion("v1.2.3"))
	})

	t.Run("falls back to the module build info", func(t *testing.T) {
		t.Parallel()

		got := resolveVersion("")
		require.NotEmpty(t, got)
		require.NotEqual(t, "0.33.0", got, "must not report a hardcoded version")
	})
}

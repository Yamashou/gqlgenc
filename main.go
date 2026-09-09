package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/gqlgo/gqlgenc/config"
	"github.com/gqlgo/gqlgenc/generator"
)

// version can be set at build time with -ldflags "-X main.version=v1.2.3".
// When it is empty, the version recorded in the module build info is used,
// which is the requested version for binaries installed with go install.
var version = ""

func main() {
	var (
		showVersion = flag.Bool("version", false, "print the version")
		configDir   = flag.String("configdir", ".", "the directory with configuration file")
	)

	flag.StringVar(configDir, "c", ".", "the directory with configuration file (shorthand)")
	flag.Parse()

	if *showVersion {
		fmt.Println(resolveVersion(version))

		return
	}

	cfg, err := config.LoadConfigFromDefaultLocations(*configDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)

		os.Exit(2)
	}

	ctx := context.Background()

	err = generator.Generate(ctx, cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)

		os.Exit(4)
	}
}

// resolveVersion returns the version to print for -version.
//
// Arguments:
//   - override: the value set at build time with -ldflags, or empty
//
// Returns:
//   - string: override when it is set, otherwise the main module version from
//     the build info, or "(devel)" when no build info is available
//
// Preconditions:
//   - none
//
// Postconditions:
//   - the result is never empty
func resolveVersion(override string) string {
	if override != "" {
		return override
	}

	info, ok := debug.ReadBuildInfo()
	if ok && info.Main.Version != "" {
		return info.Main.Version
	}

	return "(devel)"
}

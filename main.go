package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/Yamashou/gqlgenc/v3/config"
	"github.com/Yamashou/gqlgenc/v3/gen"
)

const version = "3.0.0"

var versionOption = flag.Bool("version", false, "gqlgenc version")

func main() {
	flag.Parse()
	if *versionOption {
		fmt.Printf("gqlgenc v%s", version)
		return
	}

	cfg, err := config.LoadConfigFromDefaultLocations(".")
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%+v\n", err.Error())
		os.Exit(2)
	}

	if err := gen.Generate(context.Background(), cfg); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%+v\n", err.Error())
		os.Exit(4)
	}
}

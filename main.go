package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/Yamashou/gqlgenc/v3/plugins"
	"os"

	"github.com/Yamashou/gqlgenc/v3/config"
)

const version = "3.0.0"

var versionOption = flag.Bool("version", false, "gqlgenc version")

func main() {
	flag.Parse()
	if *versionOption {
		fmt.Printf("gqlgenc v%s", version)
		return
	}

	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

}

func run() error {
	cfgFile, err := config.FindConfigFile(".")
	if err != nil {
		return err
	}

	cfg, err := config.Load(cfgFile)
	if err != nil {
		return err
	}

	ctx := context.Background()
	if err := cfg.Init(ctx); err != nil {
		return fmt.Errorf("failed to init: %w", err)
	}

	if err := plugins.Run(cfg); err != nil {
		return err
	}
	return nil
}

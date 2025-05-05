package main

import (
	"flag"
	"fmt"
	"os"
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

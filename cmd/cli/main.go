package main

import (
	"fmt"
	"os"

	"github.com/common-fate/granted/internal/build"
	"github.com/common-fate/granted/pkg/granted"

	"github.com/urfave/cli/v2"
)

// To add more commands to the CLI app, add them to pkg/commamds/entrypoint.go in GetCliApp
// this has been abstracted in rder to make testing and fish autocompletion work
func main() {
	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Printf("Granted v%s\n", build.Version)
	}
	app := granted.GetCliApp()

	//tracing.EnsureConfigured(app, "granted")

	err := app.Run(os.Args)
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}
}

package main

import (
	"fmt"
	"os"

	"github.com/common-fate/granted/internal/build"
	"github.com/common-fate/granted/pkg/granted"
	"github.com/common-fate/updatecheck"
	"github.com/fatih/color"

	"github.com/urfave/cli/v2"
)

func main() {
	updatecheck.Check(updatecheck.GrantedCLI, build.Version, !build.IsDev())
	defer updatecheck.Print()

	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Printf("Granted v%s\n", build.Version)
	}
	app := granted.GetCliApp()
	err := app.Run(os.Args)
	if err != nil {
		fmt.Fprintf(color.Error, "%s\n", err)
		os.Exit(1)
	}
}

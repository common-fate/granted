package main

import (
	"os"
	"path/filepath"

	"github.com/common-fate/clio"
	"github.com/common-fate/clio/clierr"
	"github.com/common-fate/granted/internal/build"
	"github.com/common-fate/granted/pkg/assume"
	"github.com/common-fate/granted/pkg/granted"
	"github.com/common-fate/updatecheck"
	"github.com/urfave/cli/v2"
)

func main() {
	updatecheck.Check(updatecheck.GrantedCLI, build.Version, !build.IsDev())
	defer updatecheck.Print()

	// Use a single binary to keep keychain ACLs simple, swapping behavior via argv[0]
	var app *cli.App
	switch filepath.Base(os.Args[0]) {
	case "assumego", "assumego.exe", "dassumego", "dassumego.exe":
		app = assume.GetCliApp()
	default:
		app = granted.GetCliApp()
	}

	err := app.Run(os.Args)
	if err != nil {
		// if the error is an instance of clierr.PrintCLIErrorer then print the error accordingly
		if cliError, ok := err.(clierr.PrintCLIErrorer); ok {
			cliError.PrintCLIError()
		} else {
			clio.Error(err.Error())
		}
		os.Exit(1)
	}

}

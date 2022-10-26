package main

import (
	"os"

	"github.com/common-fate/clio"
	"github.com/common-fate/clio/clierr"
	"github.com/common-fate/granted/internal/build"
	"github.com/common-fate/granted/pkg/assume"
	"github.com/common-fate/updatecheck"
)

func main() {
	updatecheck.Check(updatecheck.GrantedCLI, build.Version, !build.IsDev())
	defer updatecheck.Print()
	app := assume.GetCliApp()
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

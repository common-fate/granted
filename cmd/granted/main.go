package main

import (
	"os"

	"github.com/common-fate/clio"
	"github.com/common-fate/granted/internal/build"
	"github.com/common-fate/granted/pkg/granted"
	"github.com/common-fate/updatecheck"
)

func main() {
	updatecheck.Check(updatecheck.GrantedCLI, build.Version, !build.IsDev())
	defer updatecheck.Print()
	app := granted.GetCliApp()
	err := app.Run(os.Args)
	if err != nil {
		// if the error is an instance of clio.PrintCLIErrorer then print the error accordingly
		if clierr, ok := err.(clio.PrintCLIErrorer); ok {
			clierr.PrintCLIError()
		} else {
			clio.Error("%s", err.Error())
		}
		os.Exit(1)
	}
}

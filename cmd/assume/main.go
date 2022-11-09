package main

import (
	"os"

	"github.com/common-fate/clio"
	"github.com/common-fate/clio/clierr"
	"github.com/common-fate/granted/internal/build"
	"github.com/common-fate/granted/pkg/assume"
	"github.com/common-fate/granted/pkg/autosync"
	"github.com/common-fate/granted/pkg/granted/registry"
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

	// Note: Need to add this after app.Run() func above because in case of error we are using os.Exit(1) which will not wait for autosync to complete
	// and might cause sideeffect in the config files.
	// this should be skipped when 'granted registry' command or/and any of 'granted registry add/sync/setup/remove' subcommand is called.
	if !registry.Contains(os.Args, "registry") {
		autosync.Run()
		defer autosync.Print()
	}
}

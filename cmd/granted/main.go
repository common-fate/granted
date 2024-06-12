package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/common-fate/clio"
	"github.com/common-fate/clio/clierr"
	"github.com/common-fate/granted/internal/build"
	"github.com/common-fate/granted/pkg/assume"
	"github.com/common-fate/granted/pkg/granted"
	"github.com/common-fate/updatecheck"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

func main() {
	updatecheck.Check(updatecheck.GrantedCLI, build.Version, !build.IsDev())
	defer updatecheck.Print()

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-c
		// restore cursor in case spinner gets stuck
		// https://github.com/apppackio/apppack/commit/a711e55238af2402b4b027a73fccc663ec7ba0f4
		// https://github.com/briandowns/spinner/issues/122
		if runtime.GOOS != "windows" {
			fmt.Fprint(os.Stdin, "\033[?25h")
		}
		os.Exit(130)
	}()

	// Use a single binary to keep keychain ACLs simple, swapping behavior via argv[0]
	var app *cli.App
	switch filepath.Base(os.Args[0]) {
	case "assumego", "assumego.exe", "dassumego", "dassumego.exe":
		app = assume.GetCliApp()
	default:
		app = granted.GetCliApp()
	}

	// In development we can use FORCE_ASSUME_CLI to debug the assume command
	if os.Getenv("FORCE_ASSUME_CLI") == "true" {
		app = assume.GetCliApp()
	}

	err := app.Run(os.Args)
	if err != nil {
		// if the error is an instance of clierr.PrintCLIErrorer then print the error accordingly and exit
		// if it is a regular error, print it and then check if it wraps any clierrs and print those
		// this way we don't lose the embedded messages in the clierrs if we wrap them
		if cliError, ok := err.(clierr.PrintCLIErrorer); ok {
			cliError.PrintCLIError()
			os.Exit(1)
		} else {
			clio.Error(err.Error())
		}
		for err != nil {
			if cliError, ok := err.(clierr.PrintCLIErrorer); ok {
				cliError.PrintCLIError()
			}
			err = errors.Unwrap(err)
		}
		os.Exit(1)
	}

}

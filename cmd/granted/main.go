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

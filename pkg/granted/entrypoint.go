package granted

import (
	"fmt"
	"os"

	"github.com/common-fate/granted/internal/build"
	"github.com/urfave/cli/v2"
)

func GetCliApp() *cli.App {
	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Fprintf(os.Stderr, "Granted v%s\n", build.Version)
	}

	app := &cli.App{
		Name:                 "granted",
		Usage:                "https://granted.dev",
		UsageText:            "granted [global options] command [command options] [arguments...]",
		Version:              build.Version,
		HideVersion:          false,
		Commands:             []*cli.Command{&DefaultBrowserCommand, &CompletionCommand},
		EnableBashCompletion: true,
	}

	app.EnableBashCompletion = true

	return app
}

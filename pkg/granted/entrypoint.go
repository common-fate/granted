package granted

import (
	"fmt"
	"os"

	"github.com/common-fate/granted/internal/build"
	"github.com/common-fate/granted/pkg/banners"
	"github.com/urfave/cli/v2"
)

func GetCliApp() *cli.App {
	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Fprintf(os.Stderr, "Granted v%s\n", build.Version)
	}

	app := &cli.App{
		Flags:                []cli.Flag{&cli.BoolFlag{Name: "banner", Aliases: []string{"b"}, Usage: "Print the granted banner"}},
		Name:                 "granted",
		Usage:                "https://granted.dev",
		UsageText:            "granted [global options] command [command options] [arguments...]",
		Version:              build.Version,
		HideVersion:          false,
		Commands:             []*cli.Command{&DefaultBrowserCommand, &CompletionCommand},
		EnableBashCompletion: true,
		Before: func(c *cli.Context) error {

			if c.Bool("banner") {
				fmt.Fprintln(os.Stderr, banners.Granted())
			}
			return nil
		},
	}

	app.EnableBashCompletion = true

	return app
}

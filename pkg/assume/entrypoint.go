package assume

import (
	"fmt"
	"os"

	"github.com/common-fate/granted/internal/build"
	"github.com/common-fate/granted/pkg/alias"
	"github.com/urfave/cli/v2"
)

func GetCliApp() *cli.App {
	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Fprintf(os.Stderr, "Granted v%s\n", build.Version)
	}

	flags := []cli.Flag{
		&cli.BoolFlag{Name: "console", Aliases: []string{"c"}, Usage: "Open a web console to the role"},
		&cli.BoolFlag{Name: "extension", Aliases: []string{"e"}, Usage: "Open a web console to the role using the Granted Containers extension"},
		&cli.BoolFlag{Name: "chrome", Aliases: []string{"cr"}, Usage: "Open a web console to the role using a unique Google Chrome profile"},
		&cli.BoolFlag{Name: "verbose", Usage: "Log debug messages"},
	}

	app := &cli.App{
		Name:                 "assume",
		Usage:                "https://granted.dev",
		UsageText:            "assume [role] [account]",
		Version:              build.Version,
		HideVersion:          false,
		Flags:                flags,
		Action:               AssumeCommand,
		Commands:             []*cli.Command{&CompletionCommand},
		EnableBashCompletion: true,
		Before: func(c *cli.Context) error {

			// Setup the shell alias
			if os.Getenv("FORCE_NO_ALIAS") != "true" {
				return alias.MustBeConfigured()
			}
			return nil
		},
	}

	app.EnableBashCompletion = true

	return app
}

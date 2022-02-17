package assume

import (
	"fmt"
	"os"

	"github.com/common-fate/granted/internal/build"
	"github.com/common-fate/granted/pkg/alias"
	"github.com/common-fate/granted/pkg/browsers"
	"github.com/urfave/cli/v2"
)

func GetCliApp() *cli.App {
	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Fprintf(os.Stderr, "Granted v%s\n", build.Version)
	}

	flags := []cli.Flag{
		&cli.BoolFlag{Name: "console", Aliases: []string{"c"}, Usage: "Open a web console to the role"},
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
		EnableBashCompletion: true,
		Before: func(c *cli.Context) error {

			hasSetup, err := browsers.UserHasDefaultBrowser(c)

			if err != nil {
				return err
			}
			if !hasSetup {
				err = browsers.HandleBrowserWizard(c)
				if err != nil {
					return err
				}
			}

			if err != nil {
				return err
			}

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

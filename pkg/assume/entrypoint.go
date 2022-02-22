package assume

import (
	"fmt"
	"os"

	"github.com/common-fate/granted/internal/build"
	"github.com/common-fate/granted/pkg/alias"
	"github.com/common-fate/granted/pkg/banners"
	"github.com/common-fate/granted/pkg/browsers"
	"github.com/common-fate/granted/pkg/config"
	"github.com/common-fate/granted/pkg/debug"
	"github.com/common-fate/granted/pkg/updates"
	"github.com/urfave/cli/v2"
)

func GetCliApp() *cli.App {
	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Fprintf(os.Stderr, "Granted v%s\n", build.Version)
	}

	flags := []cli.Flag{
		&cli.BoolFlag{Name: "console", Aliases: []string{"c"}, Usage: "Open a web console to the role"},
		&cli.BoolFlag{Name: "verbose", Usage: "Log debug messages"},
		&cli.BoolFlag{Name: "banner", Aliases: []string{"b"}, Usage: "Print the assume banner"},
		&cli.StringFlag{Name: "update-checker-api-url", Value: build.UpdateCheckerApiUrl, EnvVars: []string{"UPDATE_CHECKER_API_URL"}, Hidden: true},
	}

	app := &cli.App{
		Name:                 "assume",
		Usage:                "https://granted.dev",
		UsageText:            "assume [options][Profile]",
		Version:              build.Version,
		HideVersion:          false,
		Flags:                flags,
		Action:               updates.WithUpdateCheck(AssumeCommand),
		EnableBashCompletion: true,
		Before: func(c *cli.Context) error {
			if c.Bool("verbose") {
				debug.CliVerbosity = debug.VerbosityDebug
			}

			if err := config.SetupConfigFolder(); err != nil {
				return err
			}

			if c.Bool("banner") {
				fmt.Fprintln(os.Stderr, banners.Assume())
			}

			hasSetup, err := browsers.UserHasDefaultBrowser(c)

			if err != nil {
				return err
			}
			if !hasSetup {
				err = browsers.HandleBrowserWizard(c)
				if err != nil {
					return err
				}

				//run instructions
				browsers.GrantedIntroduction()
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

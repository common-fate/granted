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
	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
)

// Prevent issues where these flags are initialised in some part of the program then used by another part
// For our use case, we need fresh copies of these flags in the app and in the assume command
// we use this to allow flags to be set on either side of the profile arg e.g `assume -c profile-name -r ap-southeast-2`
var GlobalFlags = func() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{Name: "console", Aliases: []string{"c"}, Usage: "Open a web console to the role"},
		&cli.BoolFlag{Name: "unset", Aliases: []string{"un"}, Usage: "Unset all environment variables configured by Assume"},
		&cli.BoolFlag{Name: "url", Aliases: []string{"u"}, Usage: "Get an active console session url"},
		&cli.StringFlag{Name: "service", Aliases: []string{"s"}, Usage: "Specify a service to open the console into"},
		&cli.StringFlag{Name: "region", Aliases: []string{"r"}, Usage: "Specify a region to open the console into"},
		&cli.StringSliceFlag{Name: "pass-through", Aliases: []string{"pt"}, Usage: "Pass args to proxy assumer"},
		&cli.BoolFlag{Name: "active-role", Aliases: []string{"ar"}, Usage: "Open console using active role"},
		&cli.BoolFlag{Name: "verbose", Usage: "Log debug messages"},
		&cli.StringFlag{Name: "update-checker-api-url", Value: build.UpdateCheckerApiUrl, EnvVars: []string{"UPDATE_CHECKER_API_URL"}, Hidden: true},
		&cli.StringFlag{Name: "granted-active-aws-role-profile", EnvVars: []string{"AWS_PROFILE"}, Hidden: true},
		&cli.BoolFlag{Name: "auto-configure-shell", Usage: "Configure shell alias without prompts"}}
}

func GetCliApp() *cli.App {
	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Fprintln(color.Error, banners.WithVersion(banners.Assume()))
	}

	app := &cli.App{
		Name:                 "assume",
		Writer:               color.Error,
		Usage:                "https://granted.dev",
		UsageText:            "assume [options][Profile]",
		Version:              build.Version,
		HideVersion:          false,
		Flags:                GlobalFlags(),
		Action:               updates.WithUpdateCheck(func(c *cli.Context) error { return AssumeCommand(c) }),
		EnableBashCompletion: true,
		Before: func(c *cli.Context) error {
			// unsets the exported env vars
			if c.Bool("unset") {
				err := UnsetAction(c)
				if err != nil {
					return err
				}
				os.Exit(0)
			}
			if c.Bool("verbose") {
				debug.CliVerbosity = debug.VerbosityDebug
			}

			if err := config.SetupConfigFolder(); err != nil {
				return err
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

				// run instructions
				// terminates the command with os.exit(0)
				browsers.GrantedIntroduction()
			}

			// Setup the shell alias
			if os.Getenv("FORCE_NO_ALIAS") != "true" {
				return alias.MustBeConfigured(c.Bool("auto-configure-shell"))
			}
			return nil
		},
	}

	app.EnableBashCompletion = true

	return app
}

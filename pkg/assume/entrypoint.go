package assume

import (
	"os"

	"github.com/common-fate/clio"
	"github.com/common-fate/granted/internal/build"
	"github.com/common-fate/granted/pkg/alias"
	"github.com/common-fate/granted/pkg/autosync"
	"github.com/common-fate/granted/pkg/banners"
	"github.com/common-fate/granted/pkg/browser"
	"github.com/common-fate/granted/pkg/config"
	"github.com/urfave/cli/v2"
)

// Prevent issues where these flags are initialised in some part of the program then used by another part
// For our use case, we need fresh copies of these flags in the app and in the assume command
// we use this to allow flags to be set on either side of the profile arg e.g `assume -c profile-name -r ap-southeast-2`
func GlobalFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{Name: "console", Aliases: []string{"c"}, Usage: "Open a web console to the role"},
		&cli.BoolFlag{Name: "env", Aliases: []string{"e"}, Usage: "Export credentials to a .env file"},
		&cli.BoolFlag{Name: "export", Aliases: []string{"ex"}, Usage: "Export credentials to a ~/.aws/credentials file"},
		&cli.BoolFlag{Name: "unset", Aliases: []string{"un"}, Usage: "Unset all environment variables configured by Assume"},
		&cli.BoolFlag{Name: "url", Aliases: []string{"u"}, Usage: "Get an active console session url"},
		&cli.StringFlag{Name: "service", Aliases: []string{"s"}, Usage: "Like --c, but opens to a specified service"},
		&cli.StringFlag{Name: "region", Aliases: []string{"r"}, Usage: "region to launch the console or export to the terminal"},
		&cli.StringSliceFlag{Name: "pass-through", Aliases: []string{"pt"}, Usage: "Pass args to proxy assumer"},
		&cli.BoolFlag{Name: "active-role", Aliases: []string{"ar"}, Usage: "Open console using active role"},
		&cli.BoolFlag{Name: "verbose", Usage: "Log debug messages"},
		&cli.StringFlag{Name: "update-checker-api-url", Value: build.UpdateCheckerApiUrl, EnvVars: []string{"UPDATE_CHECKER_API_URL"}, Hidden: true},
		&cli.StringFlag{Name: "active-aws-profile", EnvVars: []string{"AWS_PROFILE"}, Hidden: true},
		&cli.BoolFlag{Name: "auto-configure-shell", Usage: "Configure shell alias without prompts"},
		&cli.StringFlag{Name: "exec", Usage: "Assume a profile then execute this command"},
		&cli.StringFlag{Name: "duration", Aliases: []string{"d"}, Usage: "Set session duration for your assumed role"},
		&cli.BoolFlag{Name: "sso", Usage: "Assume an account and role with provided SSO flags"},
		&cli.StringFlag{Name: "sso-start-url", Usage: "Use this in conjunction with --sso, the sso-start-url"},
		&cli.StringFlag{Name: "sso-region", Usage: "Use this in conjunction with --sso, the sso-region"},
		&cli.StringFlag{Name: "account-id", Usage: "Use this in conjunction with --sso, the account-id"},
		&cli.StringFlag{Name: "role-name", Usage: "Use this in conjunction with --sso, the role-name"},
	}
}

func GetCliApp() *cli.App {
	cli.VersionPrinter = func(c *cli.Context) {
		clio.Log(banners.WithVersion(banners.Assume()))
	}

	app := &cli.App{
		Name:                 "assume",
		Writer:               os.Stderr,
		Usage:                "https://granted.dev",
		UsageText:            "assume [options][Profile]",
		Version:              build.Version,
		HideVersion:          false,
		Flags:                GlobalFlags(),
		Action:               AssumeCommand,
		EnableBashCompletion: true,
		BashComplete:         Completion,
		Before: func(c *cli.Context) error {

			// unsets the exported env vars
			if c.Bool("unset") {
				err := UnsetAction(c)
				if err != nil {
					return err
				}
				os.Exit(0)
			}

			clio.SetLevelFromEnv("GRANTED_LOG")
			if c.Bool("verbose") {
				clio.SetLevelFromString("debug")
			}
			err := ValidateSSOFlags(c)
			if err != nil {
				return err
			}

			if err := config.SetupConfigFolder(); err != nil {
				return err
			}

			hasSetup, err := browser.UserHasDefaultBrowser(c)
			if err != nil {
				return err
			}
			if !hasSetup {
				browserName, err := browser.HandleBrowserWizard(c)
				if err != nil {
					return err
				}

				//see if they want to set their sso browser the same as their granted default
				err = browser.SSOBrowser(browserName)
				if err != nil {
					return err
				}

				// run instructions
				// terminates the command with os.exit(0)
				browser.GrantedIntroduction()
			}
			// Sync granted profile registries if enabled
			autosync.Run()

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

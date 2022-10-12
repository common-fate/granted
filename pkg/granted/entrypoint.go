package granted

import (
	"fmt"

	"github.com/common-fate/granted/internal/build"
	"github.com/common-fate/granted/pkg/banners"
	"github.com/common-fate/granted/pkg/config"
	"github.com/common-fate/granted/pkg/debug"
	"github.com/common-fate/granted/pkg/granted/settings"
	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
)

func GetCliApp() *cli.App {
	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Fprintln(color.Error, banners.WithVersion(banners.Granted()))
	}

	flags := []cli.Flag{
		&cli.BoolFlag{Name: "verbose", Usage: "Log debug messages"},
		&cli.StringFlag{Name: "update-checker-api-url", Value: build.UpdateCheckerApiUrl, EnvVars: []string{"UPDATE_CHECKER_API_URL"}, Hidden: true},
	}

	app := &cli.App{
		Flags:                flags,
		Name:                 "granted",
		Usage:                "https://granted.dev",
		UsageText:            "granted [global options] command [command options] [arguments...]",
		Version:              build.Version,
		HideVersion:          false,
		Commands:             []*cli.Command{&DefaultBrowserCommand, &settings.SettingsCommand, &CompletionCommand, &TokenCommand, &SSOTokensCommand, &UninstallCommand, &SSOCommand, &CredentialsCommand, &CredentialProcess},
		EnableBashCompletion: true,
		Before: func(c *cli.Context) error {
			if c.Bool("verbose") {
				debug.CliVerbosity = debug.VerbosityDebug
			}
			if err := config.SetupConfigFolder(); err != nil {
				return err
			}
			return nil
		},
	}

	return app
}

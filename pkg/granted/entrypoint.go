package granted

import (
	"path"

	"github.com/common-fate/cli/cmd/command"
	"github.com/common-fate/clio"
	"github.com/common-fate/clio/cliolog"
	"github.com/common-fate/granted/internal/build"
	"github.com/common-fate/granted/pkg/banners"
	"github.com/common-fate/granted/pkg/config"
	"github.com/common-fate/granted/pkg/granted/exp"
	"github.com/common-fate/granted/pkg/granted/middleware"
	"github.com/common-fate/granted/pkg/granted/registry"
	"github.com/common-fate/granted/pkg/granted/settings"
	"github.com/common-fate/granted/pkg/securestorage"
	"github.com/common-fate/useragent"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

func GetCliApp() *cli.App {
	cli.VersionPrinter = func(c *cli.Context) {
		clio.Log(banners.WithVersion(banners.Granted()))
	}

	flags := []cli.Flag{
		&cli.BoolFlag{Name: "verbose", Usage: "Log debug messages"},
		&cli.StringFlag{Name: "update-checker-api-url", Value: build.UpdateCheckerApiUrl, EnvVars: []string{"UPDATE_CHECKER_API_URL"}, Hidden: true},
	}

	app := &cli.App{
		Flags:       flags,
		Name:        "granted",
		Usage:       "https://granted.dev",
		UsageText:   "granted [global options] command [command options] [arguments...]",
		Version:     build.Version,
		HideVersion: false,
		Commands: []*cli.Command{
			&DefaultBrowserCommand,
			&settings.SettingsCommand,
			&CompletionCommand,
			&TokenCommand,
			&SSOTokensCommand,
			&UninstallCommand,
			&SSOCommand,
			&CredentialsCommand,
			middleware.WithBeforeFuncs(&CredentialProcess, middleware.WithAutosync()),
			&registry.ProfileRegistryCommand,
			&ConsoleCommand,
			&login,
			&exp.Command,
			&CacheCommand,
		},
		EnableBashCompletion: true,
		Before: func(c *cli.Context) error {
			clio.SetLevelFromEnv("GRANTED_LOG")

			grantedFolder, err := config.GrantedConfigFolder()
			if err != nil {
				return err
			}

			logfilepath := path.Join(grantedFolder, "log")

			clio.SetFileLogging(cliolog.FileLoggerConfig{
				Filename: logfilepath,
			})

			zap.ReplaceGlobals(clio.G())
			if c.Bool("verbose") {
				clio.SetLevelFromString("debug")
			}
			if err := config.SetupConfigFolder(); err != nil {
				return err
			}
			// set the user agent
			c.Context = useragent.NewContext(c.Context, "granted", build.Version)

			return nil
		},
	}

	return app
}

var login = cli.Command{
	Name:  "login",
	Usage: "Log in to Common Fate",
	Flags: []cli.Flag{
		&cli.BoolFlag{Name: "lazy", Usage: "When the lazy flag is used, a login flow will only be started when the access token is expired"},
	},
	Action: func(c *cli.Context) error {
		k, err := securestorage.NewCF().Storage.Keyring()
		if err != nil {
			return errors.Wrap(err, "loading keyring")
		}

		// wrap the nested CLI command with the keyring
		lf := command.LoginFlow{Keyring: k}

		return lf.LoginAction(c)
	},
}

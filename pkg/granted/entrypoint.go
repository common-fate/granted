package granted

import (
	"fmt"
	"os"

	"github.com/common-fate/granted/internal/build"
	"github.com/common-fate/granted/pkg/assume"
	"github.com/urfave/cli/v2"
)

func GetCliApp() *cli.App {
	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Fprintf(os.Stderr, "Granted v%s\n", build.Version)
	}

	app := &cli.App{
		Name:        "granted",
		Usage:       "https://granted.dev",
		UsageText:   "assume [role] [account]",
		Version:     build.Version,
		HideVersion: false,
		//Action:               AssumeCommand,
		Commands:             []*cli.Command{&DefaultBrowserCommand},
		EnableBashCompletion: true,
		Before: func(c *cli.Context) error {

			hasSetup, err := assume.UserHasDefaultBrowser(c)

			if err != nil {
				return err
			}
			if !hasSetup {
				err = assume.HandleBrowserWizard(c)
				if err != nil {
					return err
				}
			}

			if err != nil {
				return err
			}

			//Setup the shell alias
			// if os.Getenv("FORCE_NO_ALIAS") != "true" {
			// 	return alias.MustBeConfigured()
			// }
			return nil
		},
	}

	app.EnableBashCompletion = true

	return app
}

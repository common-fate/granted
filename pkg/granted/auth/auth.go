package auth

import (
	"strings"

	"github.com/common-fate/cli/cmd/cli/command"
	"github.com/common-fate/clio"
	"github.com/common-fate/sdk/config"
	"github.com/common-fate/sdk/loginflow"
	"github.com/urfave/cli/v2"
)

var Command = cli.Command{
	Name:  "auth",
	Usage: "Manage OIDC authentication for Granted",
	Flags: []cli.Flag{},
	Subcommands: []*cli.Command{
		&command.Configure,
		&loginCommand,
		&logoutCommand,
		&command.Context,
	},
}

var loginCommand = cli.Command{
	Name:  "login",
	Usage: "Authenticate to an OIDC provider",
	Action: func(c *cli.Context) error {
		cfg, err := config.LoadDefault(c.Context)

		if err != nil && strings.Contains(err.Error(), "config file does not exist") {
			clio.Debugw("prompting user login because token is expired", "error_details", err.Error())
			// NOTE(chrnorm): ideally we'll bubble up a more strongly typed error in future here, to avoid the string comparison on the error message.

			// the OAuth2.0 token is expired so we should prompt the user to log in
			clio.Infof("Config file not found. To get set up with Common Fate run `granted auth configure https://cf.demo.io`")

		}
		if err != nil {
			return err
		}

		lf := loginflow.NewFromConfig(cfg)

		return lf.Login(c.Context)
	},
}

var logoutCommand = cli.Command{
	Name:  "logout",
	Usage: "Log out of an OIDC provider",
	Action: func(c *cli.Context) error {
		cfg, err := config.LoadDefault(c.Context)
		if err != nil {
			return err
		}

		return cfg.TokenStore.Clear()
	},
}

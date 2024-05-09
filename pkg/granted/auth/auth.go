package auth

import (
	"github.com/common-fate/cli/cmd/cli/command"
	"github.com/urfave/cli/v2"
)

var Command = cli.Command{
	Name:  "auth",
	Usage: "Manage OIDC authentication for Granted",
	Flags: []cli.Flag{},
	Subcommands: []*cli.Command{
		&command.Configure,
		&command.Login,
		&command.Logout,
		&command.Context,
	},
}

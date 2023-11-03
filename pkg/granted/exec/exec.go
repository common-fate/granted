package exec

import "github.com/urfave/cli/v2"

var Command = cli.Command{
	Name:  "exec",
	Usage: "Execute a command against a particular cloud role or resource",
	Subcommands: []*cli.Command{
		&gkeCommand,
	},
}

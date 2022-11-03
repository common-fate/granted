package registry

import "github.com/urfave/cli/v2"

var ProfileRegistry = cli.Command{
	Name:        "registry",
	Subcommands: []*cli.Command{&AddCommand, &SyncCommand},
}

package registry

import "github.com/urfave/cli/v2"

var ProfileRegistry = cli.Command{
	Name:        "registry",
	Description: "Add git repository of AWS profiles which will be synced to ~/.aws/config file",
	Subcommands: []*cli.Command{&AddCommand, &SyncCommand, &RemoveCommand},
}

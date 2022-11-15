package registry

import (
	"github.com/common-fate/clio"
	grantedConfig "github.com/common-fate/granted/pkg/config"
	"github.com/urfave/cli/v2"
)

var ProfileRegistry = cli.Command{
	Name:        "registry",
	Usage:       "Manage Profile Registries",
	Description: "Profile Registries allow you to easily share AWS profile configuration in a team.",
	Subcommands: []*cli.Command{&SetupCommand, &AddCommand, &SyncCommand, &RemoveCommand},
	Action: func(c *cli.Context) error {
		gConf, err := grantedConfig.Load()
		if err != nil {
			return err
		}

		if len(gConf.ProfileRegistryURLS) == 0 {
			clio.Warn("You haven't connected any Profile Registries yet.")
			clio.Info("Connect to a Profile Registry by running 'granted registry add <your_repo>'")
			return nil
		}

		clio.Info("Granted is currently synced with following registries:")
		for i, url := range gConf.ProfileRegistryURLS {
			clio.Logf("\t %d: %s", (i + 1), url)
		}
		clio.NewLine()

		clio.Info("To add new registry use 'granted registry add <your_repo>'")
		clio.Info("To remove a registry use 'granted registry remove' and select from the options")
		clio.Info("To sync a registry use 'granted registry sync'")

		return nil
	},
}

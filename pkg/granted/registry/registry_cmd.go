package registry

import (
	"github.com/common-fate/clio"
	"github.com/urfave/cli/v2"
)

var ProfileRegistryCommand = cli.Command{
	Name:        "registry",
	Usage:       "Manage Profile Registries",
	Description: "Profile Registries allow you to easily share AWS profile configuration in a team.",
	Subcommands: []*cli.Command{&AddCommand, &SyncCommand, &RemoveCommand, &MigrateCommand, &SetupCommand},
	Action: func(c *cli.Context) error {
		registries, err := GetProfileRegistries()
		if err != nil {
			return err
		}

		if len(registries) == 0 {
			clio.Warn("You haven't connected any Profile Registries yet.")
			clio.Info("Connect to a Profile Registry by running 'granted registry add -n <registry_name> -u <your_repo>'")
			return nil
		}

		clio.Info("Granted is currently synced with following registries:")
		for i, r := range registries {
			clio.Logf("\t %d: %s with name '%s'", (i + 1), r.Config.URL, r.Config.Name)
		}
		clio.NewLine()

		clio.Info("To add new registry use 'granted registry add <your_repo>'")
		clio.Info("To remove a registry use 'granted registry remove' and select from the options")
		clio.Info("To sync a registry use 'granted registry sync'")

		return nil
	},
}

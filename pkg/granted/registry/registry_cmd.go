package registry

import (
	"github.com/common-fate/clio"
	grantedConfig "github.com/common-fate/granted/pkg/config"
	"github.com/urfave/cli/v2"
)

var ProfileRegistry = cli.Command{
	Name:        "registry",
	Description: "Add git repository of AWS profiles which will be synced to ~/.aws/config file",
	Subcommands: []*cli.Command{&SetupCommand, &AddCommand, &SyncCommand, &RemoveCommand},
	Action: func(c *cli.Context) error {

		gConf, err := grantedConfig.Load()
		if err != nil {
			return err
		}

		if len(gConf.ProfileRegistryURLS) <= 0 {
			clio.Info("There are no registry subscribed to granted yet.\n Use 'granted registry add <your_repo>' to configure subscription.\n")

			return nil
		}

		clio.Log("Granted is currently synced with following registries: ")
		for i, url := range gConf.ProfileRegistryURLS {
			clio.Logf("\t %d: %s", (i + 1), url)
		}

		clio.Infoln("To add new registry use 'granted registry add <your_repo>'")
		clio.Infoln("To remove registry use 'granted registry remove' and select from the options")
		clio.Infoln("To sync registry use 'granted registry sync'")

		return nil
	},
}

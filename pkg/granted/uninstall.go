package granted

import (
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/common-fate/clio"
	"github.com/common-fate/granted/pkg/alias"
	"github.com/common-fate/granted/pkg/config"
	"github.com/urfave/cli/v2"
)

var UninstallCommand = cli.Command{
	Name:  "uninstall",
	Usage: "Remove all Granted configuration",
	Action: func(c *cli.Context) error {
		withStdio := survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)
		in := &survey.Confirm{
			Message: "Are you sure you want to remove your Granted config?",
			Default: true,
		}
		var confirm bool
		err := survey.AskOne(in, &confirm, withStdio)
		if err != nil {
			return err
		}
		if confirm {

			err = alias.UninstallDefaultShellAlias()
			if err != nil {
				clio.Error("%s", err)
			}
			grantedFolder, err := config.GrantedConfigFolder()
			if err != nil {
				return err
			}
			err = os.RemoveAll(grantedFolder)
			if err != nil {
				return err
			}

			clio.Success("Removed Granted config folder %s\n", grantedFolder)
		}
		return nil
	},
}

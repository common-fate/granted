package granted

import (
	"fmt"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/common-fate/granted/pkg/alias"
	"github.com/common-fate/granted/pkg/config"
	"github.com/fatih/color"
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

			_, err = alias.UninstallDefaultShellAlias()
			if err != nil {
				fmt.Fprintln(color.Error, err)
			}
			grantedFolder, err := config.GrantedConfigFolder()
			if err != nil {
				return err
			}
			err = os.RemoveAll(grantedFolder)
			if err != nil {
				return err
			}

			fmt.Printf("removed Granted config folder %s\n", grantedFolder)
			fmt.Fprintln(color.Error, "[✔] all Granted config has been removed")
		}
		return nil
	},
}

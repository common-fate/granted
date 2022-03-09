package granted

import (
	"fmt"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/common-fate/granted/pkg/config"
	"github.com/common-fate/granted/pkg/credstore"
	"github.com/urfave/cli/v2"
)

var ClearCommand = cli.Command{
	Name:        "reset",
	Usage:       "Factory reset of granted config",
	Subcommands: []*cli.Command{},
	Action: func(ctx *cli.Context) error {

		withStdio := survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)
		in := &survey.Confirm{
			Message: "Are you sure you want to reset your Granted config?",
			Default: true,
		}
		var confirm bool
		err := survey.AskOne(in, &confirm, withStdio)
		if err != nil {
			return err
		}
		if confirm {

			err := credstore.ClearAll()
			if err != nil {
				return err
			}
			grantedFolder, err := config.GrantedConfigFolder()
			if err != nil {
				return err
			}
			os.RemoveAll(grantedFolder)
			fmt.Fprintf(os.Stderr, "Cleared all granted config")
		} else {
			fmt.Fprintf(os.Stderr, "Exited reset flow")
		}

		return nil

	},
}

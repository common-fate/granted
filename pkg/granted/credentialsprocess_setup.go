package granted

import (
	"errors"
	"fmt"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/urfave/cli/v2"
)

var ConfigSetup = cli.Command{
	Name:  "setup",
	Usage: "Alters your AWS Config credential_processs",
	Action: func(c *cli.Context) error {

		fmt.Println("This will override the `credential_process` in your config file.")
		withStdio := survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)
		in := &survey.Confirm{
			Message: "Would you like to proceed?",
			Default: false,
		}
		var confirm bool
		err := survey.AskOne(in, &confirm, withStdio)
		if err != nil {
			return err
		}
		if !confirm {
			return errors.New("cancelled alias installation")
		}

		// load the default config

		// Itterate over each section

		// If a section contains a non-standard credential_process

		// Prompt the user for confrimation

		// Write to credential_process our custom script

		// Done :)

		return nil
	},
}

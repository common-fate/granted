package granted

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
)

var CredentialsProcess = cli.Command{
	Name:   "credentialsprocess",
	Usage:  "",
	Hidden: true,
	Flags:  []cli.Flag{&cli.StringFlag{Name: "profile"}},
	Action: func(c *cli.Context) error {

		// Check if the session can be assumed

		// If yes, run standard `aws aws-sso-credential-process...`

		// Else log a message to the console

		fmt.Fprintln(color.Error, color.GreenString("[✔] test"))
		os.Exit(1)

		// Try use this and pipe the output
		err = exec.Command("echo", "TEST").Run()

		if err != nil {
			return err
		}

		// c.App.ErrWriter

		return nil
	},
}

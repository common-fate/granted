package granted

import (
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/common-fate/granted/pkg/iamcredstore"
	"github.com/common-fate/granted/pkg/testable"
	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
)

var GvaultCommand = cli.Command{
	Name:        "gvault",
	Usage:       "Manage aws access tokens",
	Subcommands: []*cli.Command{&GvaultCommandAddCommand},
	Action:      TokenListCommand.Action,
}

var GvaultCommandAddCommand = cli.Command{
	Name:  "add",
	Usage: "Adds a new credential",
	Action: func(c *cli.Context) error {
		withStdio := survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)

		profileName := c.Args().First()
		if profileName == "" {
			ibpIn := survey.Input{Message: "Enter a name for profile"}
			err := testable.AskOne(&ibpIn, &profileName, withStdio)
			if err != nil {
				return err
			}
		}

		var accessKey string
		ac := survey.Input{Message: "Access Key: "}
		err := testable.AskOne(&ac, &accessKey, withStdio)
		if err != nil {
			return err
		}
		var accessSecret string
		as := survey.Password{Message: "Secret Key: "}
		err = testable.AskOne(&as, &accessSecret, withStdio)
		if err != nil {
			return err
		}

		creds := iamcredstore.IAMUserCredentials{
			ProfileName:     profileName,
			AccessKeyID:     accessKey,
			SecretAccessKey: accessSecret,
		}

		err = iamcredstore.Store(profileName, creds)
		if err != nil {
			return err
		}

		var outcreds iamcredstore.IAMUserCredentials

		out, err := iamcredstore.Retrieve(profileName, outcreds)
		if err != nil {
			return err
		}

		green := color.New(color.FgGreen)

		green.Fprintf(color.Error, "\nCreds set: %s\n", out)
		return nil
	},
}

package granted

import (
	"fmt"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/common-fate/granted/pkg/config"
	"github.com/common-fate/granted/pkg/credstore"
	"github.com/common-fate/granted/pkg/testable"
	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
)

var CredstoreCommand = cli.Command{
	Name:        "credstore",
	Usage:       "Manage where IAM credentials are stored",
	Subcommands: []*cli.Command{&SetIAMStoreCommand, &WhichIAMStoreCommand, &AddIAMStoreCommand},
	Action:      TokenListCommand.Action,
}

var WhichIAMStoreCommand = cli.Command{
	Name:  "which",
	Usage: "If set will return the credstore type ",
	Action: func(ctx *cli.Context) error {

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		fmt.Printf(*cfg.IAMCredStore)
		return nil
	},
}

var SetIAMStoreCommand = cli.Command{
	Name:  "set",
	Usage: "Set the credstore location",
	Action: func(ctx *cli.Context) error {

		fmt.Printf("You are currently storing your credentials in: ")

		storeList := []string{"Default", "Keychain"}
		withStdio := survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)
		in := survey.Select{
			Message: "Select location to store IAM credentials",
			Options: storeList,
		}
		fmt.Fprintln(os.Stderr)
		var out string
		err := testable.AskOne(&in, &out, withStdio)
		if err != nil {
			return err
		}

		cfg, err := config.Load()

		if err != nil {
			return err
		}
		cfg.IAMCredStore = &out

		cfg.Save()

		switch out {
		case "Default":
			fmt.Printf("IAM creds will be saved and read from ~/.aws/credentials file")
		case "Keychain":
			fmt.Printf("IAM creds will be saved and read from your devices keychain")
		}

		return nil
	},
}

var AddIAMStoreCommand = cli.Command{
	Name:  "add",
	Usage: "Add new set of IAM credentials to your credfile",
	Action: func(ctx *cli.Context) error {
		cfg, err := config.Load()

		if err != nil {
			return err
		}

		creds := aws.Credentials{}

		fmt.Printf("You are currently storing your credentials in: %s\n", *cfg.IAMCredStore)

		withStdio := survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)

		var profile string
		in := survey.Input{Message: "Profile Name: "}
		fmt.Fprintln(color.Error)
		err = testable.AskOne(&in, &profile, withStdio)
		if err != nil {
			return err
		}

		in1 := survey.Input{Message: "Access Key Id: "}
		fmt.Fprintln(color.Error)
		err = testable.AskOne(&in1, &creds.AccessKeyID, withStdio)
		if err != nil {
			return err
		}

		in2 := survey.Password{Message: "Secret Key: "}
		fmt.Fprintln(color.Error)
		err = testable.AskOne(&in2, &creds.SecretAccessKey, withStdio)
		if err != nil {
			return err
		}

		err = credstore.Store(profile, creds)
		if err != nil {
			return err
		}

		fmt.Printf("Saved %s to Credfile: %s", profile, *cfg.IAMCredStore)

		return nil
	},
}

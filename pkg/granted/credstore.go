package granted

import (
	"fmt"
	"os"
	"time"

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

		if cfg.UseSecureCredStorage {
			fmt.Printf("Credentials are saved and read from a secure local keychain")
		} else {
			fmt.Printf("Credentials are saved and read from a `~/.aws/credentials`")
		}

		return nil
	},
}

var SetIAMStoreCommand = cli.Command{
	Name:  "set",
	Usage: "Set the credstore location",
	Action: func(ctx *cli.Context) error {

		fmt.Printf("You are currently storing your credentials in: ")

		withStdio := survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)
		in := &survey.Confirm{
			Message: "Store credentials in your local keychain?",
			Default: true,
		}
		var confirm bool
		err := survey.AskOne(in, &confirm, withStdio)
		if err != nil {
			return err
		}

		cfg, err := config.Load()

		if err != nil {
			return err
		}
		cfg.UseSecureCredStorage = confirm

		err = cfg.Save()
		if err != nil {
			return err
		}
		if cfg.UseSecureCredStorage {
			fmt.Printf("IAM creds will be saved and read from your devices keychain")
		} else {
			fmt.Printf("IAM creds will be saved and read from ~/.aws/credentials file")

		}

		return nil
	},
}

var AddIAMStoreCommand = cli.Command{
	Name:  "add",
	Usage: "Add new set of IAM credentials to your credfile",
	Action: func(ctx *cli.Context) error {

		//only allow this functionality if cfg.

		cfg, err := config.Load()

		if err != nil {
			return err
		}

		if !cfg.UseSecureCredStorage {
			fmt.Printf("Adding IAM credentials to `~/.aws/credentials is not yet supported by granted. UseSecureCredStorage: %t", cfg.UseSecureCredStorage)
			return nil
		}

		creds := aws.Credentials{}

		if cfg.UseSecureCredStorage {
			fmt.Printf("IAM creds are being saved and read from your devices keychain")
		} else {
			fmt.Printf("IAM creds are being saved and read from ~/.aws/credentials file")

		}
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

		creds.CanExpire = true
		creds.Expires = time.Now().Add(time.Minute * 60)

		err = credstore.Store(profile, creds)
		if err != nil {
			return err
		}

		fmt.Printf("Saved %s to Credfile", profile)

		return nil
	},
}

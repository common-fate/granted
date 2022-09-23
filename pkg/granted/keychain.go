package granted

import (
	"fmt"
	"os"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/common-fate/granted/pkg/cfaws"
	"github.com/common-fate/granted/pkg/credstore"
	"github.com/common-fate/granted/pkg/testable"
	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
)

var KeychainCommand = cli.Command{
	Name:        "keychain",
	Usage:       "Manage where IAM credentials are stored",
	Subcommands: []*cli.Command{&AddIAMStoreCommand, &ImportIAMStoreCommand},
	Action:      TokenListCommand.Action,
}

var AddIAMStoreCommand = cli.Command{
	Name:  "add",
	Usage: "Add new set of IAM credentials to your keychain",
	Action: func(ctx *cli.Context) error {
		var profile string

		args := ctx.Args()

		//check if first argument is passed as a cred name
		if args.First() != "add" {
			profile = args.First()
		}

		withStdio := survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)

		if profile != "" {
			in := survey.Input{Message: "Profile Name: "}
			fmt.Fprintln(color.Error)
			err := testable.AskOne(&in, &profile, withStdio)
			if err != nil {
				return err
			}
		}

		var creds aws.Credentials

		in1 := survey.Input{Message: "Access Key Id: "}
		fmt.Fprintln(color.Error)
		err := testable.AskOne(&in1, &creds.AccessKeyID, withStdio)
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

		fmt.Printf("Saved %s to keychain", profile)

		return nil
	},
}

var ImportIAMStoreCommand = cli.Command{
	Name:  "import",
	Usage: "Import credentials from ~/.credentials file into keychain",
	Action: func(ctx *cli.Context) error {
		var profile string
		//read the credentials file

		args := ctx.Args()

		if args.First() != "" {
			profile = args.First()
		}

		profiles, err := cfaws.LoadProfiles()
		if err != nil {
			return err
		}

		//find the profile
		iamProfile, err := profiles.LoadInitialisedProfile(ctx.Context, profile)
		if err != nil {
			return err
		}
		//add creds to the keychain
		creds := iamProfile.AWSConfig.Credentials

		creds.CanExpire = true
		creds.Expires = time.Now().Add(time.Minute * 60)

		err = credstore.Store(profile, creds)
		if err != nil {
			return err
		}

		//remove creds from credfile
		err = profiles.RemoveProfileFromCredentials(profile)
		if err != nil {
			return err
		}

		fmt.Printf("Saved %s to keychain", profile)

		return nil
	},
}

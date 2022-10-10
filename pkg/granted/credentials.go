package granted

import (
	"errors"
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"

	"github.com/common-fate/granted/pkg/cfaws"
	"github.com/common-fate/granted/pkg/securestorage"
	"github.com/common-fate/granted/pkg/testable"
	"github.com/urfave/cli/v2"
)

var CredentialsCommand = cli.Command{
	Name:        "credentials",
	Usage:       "Manage secure IAM credentials",
	Subcommands: []*cli.Command{&AddCredentialsCommand, &ImportCredentialsCommand},
}

var AddCredentialsCommand = cli.Command{
	Name:  "add",
	Usage: "Add IAM credentials to secure storage",
	Action: func(c *cli.Context) error {
		profile := c.Args().First()
		if profile == "" {
			in := survey.Input{Message: "Profile Name: "}
			fmt.Println()
			err := testable.AskOne(&in, &profile)
			if err != nil {
				return err
			}
		}

		var creds aws.Credentials
		in1 := survey.Input{Message: "Access Key Id: "}
		fmt.Println()
		err := testable.AskOne(&in1, &creds.AccessKeyID)
		if err != nil {
			return err
		}

		in2 := survey.Password{Message: "Secret Access Key: "}
		fmt.Println()
		err = testable.AskOne(&in2, &creds.SecretAccessKey)
		if err != nil {
			return err
		}
		secureIAMCredentialStorage := securestorage.NewSecureIAMCredentialStorage()
		err = secureIAMCredentialStorage.StoreCredentials(profile, creds)
		if err != nil {
			return err
		}

		fmt.Printf("Saved %s to secure storage", profile)

		return nil
	},
}

var ImportCredentialsCommand = cli.Command{
	Name:  "import",
	Usage: "Import credentials from ~/.credentials file into secure storage",
	Action: func(c *cli.Context) error {
		profileName := c.Args().First()
		// profiles, err := cfaws.LoadProfiles()
		// if err != nil {
		// 	return err
		// }

		profile, err := config.LoadSharedConfigProfile(c.Context, profileName, func(lsco *config.LoadSharedConfigOptions) {
			// Don't load from the config file
			lsco.ConfigFiles = []string{}
		})
		if !(profile.Credentials.HasKeys() && profile.Credentials.SessionToken == "") {
			return errors.New("profile is not valid")
		}
		secureIAMCredentialStorage := securestorage.NewSecureIAMCredentialStorage()
		err = secureIAMCredentialStorage.StoreCredentials(profileName, profile.Credentials)
		if err != nil {
			return err
		}

		//remove creds from credfile
		err = cfaws.RemoveProfileFromCredentialsFile(profileName)
		if err != nil {
			return err
		}

		fmt.Printf("Saved %s to keychain", profile)

		return nil
	},
}

package granted

import (
	"fmt"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/bigkevmcd/go-configparser"
	"github.com/pkg/errors"

	"github.com/common-fate/granted/internal/build"
	"github.com/common-fate/granted/pkg/cfaws"
	"github.com/common-fate/granted/pkg/securestorage"
	"github.com/common-fate/granted/pkg/testable"
	"github.com/urfave/cli/v2"
)

var CredentialsCommand = cli.Command{
	Name:        "credentials",
	Usage:       "Manage secure IAM credentials",
	Subcommands: []*cli.Command{&AddCredentialsCommand, &ImportCredentialsCommand, &UpdateCredentialsCommand, &ListCredentialsCommand, &ClearCredentialsCommand},
}

var AddCredentialsCommand = cli.Command{
	Name:  "add",
	Usage: "Add IAM credentials to secure storage",
	Action: func(c *cli.Context) error {
		profileName := c.Args().First()
		if profileName == "" {
			in := survey.Input{Message: "Profile Name: "}
			fmt.Println()
			err := testable.AskOne(&in, &profileName)
			if err != nil {
				return err
			}
		}
		profiles, err := cfaws.LoadProfiles()
		if err != nil {
			return err
		}

		if profiles.HasProfile(profileName) {
			return fmt.Errorf("profile with name %s already exists", profileName)
		}
		var creds aws.Credentials
		in1 := survey.Input{Message: "Access Key Id: "}
		fmt.Println()
		err = testable.AskOne(&in1, &creds.AccessKeyID)
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
		err = secureIAMCredentialStorage.StoreCredentials(profileName, creds)
		if err != nil {
			return err
		}
		creds, err = secureIAMCredentialStorage.GetCredentials(profileName)
		if err != nil {
			return err
		}
		configPath := config.DefaultSharedConfigFilename()
		configFile, err := configparser.NewConfigParserFromFile(configPath)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		sectionName := "profile " + profileName
		if err := configFile.AddSection(sectionName); err != nil {
			return err
		}

		err = configFile.Set(sectionName, "credential_process", fmt.Sprintf("%s credential-process --profile=%s", build.GrantedBinaryName(), profileName))
		if err != nil {
			return err
		}
		err = configFile.SaveWithDelimiter(configPath, "=")
		if err != nil {
			return err
		}
		fmt.Printf("Saved %s to secure storage", profileName)

		return nil
	},
}

var ImportCredentialsCommand = cli.Command{
	Name:  "import",
	Usage: "Import credentials from ~/.credentials file into secure storage",
	Action: func(c *cli.Context) error {
		profileName := c.Args().First()
		if profileName == "" {
			in := survey.Input{Message: "Profile Name: "}
			fmt.Println()
			err := testable.AskOne(&in, &profileName)
			if err != nil {
				return err
			}
		}
		profiles, err := cfaws.LoadProfiles()
		if err != nil {
			return err
		}
		if !profiles.HasProfile(profileName) {
			return fmt.Errorf("profile with name %s does not exist", profileName)
		}
		profile, err := profiles.LoadInitialisedProfile(c.Context, profileName)
		if err != nil {
			return err
		}
		// @TODO: we can provide some better messaging here by checking for parent profiles etc, if the root profile in this profiles chain is an IAM profile with plain text keys, we shoudl promote adding that one instead
		if !profile.AWSConfig.Credentials.HasKeys() {
			return fmt.Errorf("profile %s does not have IAM credentials", profileName)
		}
		secureIAMCredentialStorage := securestorage.NewSecureIAMCredentialStorage()
		err = secureIAMCredentialStorage.StoreCredentials(profileName, profile.AWSConfig.Credentials)
		if err != nil {
			return err
		}

		//fetch parsed credentials file
		credsPath := config.DefaultSharedCredentialsFilename()
		credsFile, err := configparser.NewConfigParserFromFile(credsPath)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}

		err = credsFile.RemoveSection(profileName)
		if err != nil {
			return err
		}
		configFilename := config.DefaultSharedCredentialsFilename()

		err = credsFile.SaveWithDelimiter(configFilename, "=")
		if err != nil {
			return err
		}
		configPath := config.DefaultSharedConfigFilename()
		configFile, err := configparser.NewConfigParserFromFile(configPath)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		sectionName := "profile " + profileName
		if !configFile.HasSection(sectionName) {
			if err := configFile.AddSection(sectionName); err != nil {
				return err
			}
		}

		err = configFile.Set(sectionName, "credential_process", fmt.Sprintf("%s credential-process --profile=%s", build.GrantedBinaryName(), profileName))
		if err != nil {
			return err
		}

		// @TODO: I need to do a full clone of the profile both from credentials file and config, need to investigate how attributes are merged by the aws cli
		// for k,v := range profile.RawConfig{

		// }
		if profile.AWSConfig.Region != "" {
			err = configFile.Set(sectionName, "region", profile.AWSConfig.Region)
			if err != nil {
				return err
			}
		}
		err = configFile.SaveWithDelimiter(configPath, "=")
		if err != nil {
			return err
		}
		fmt.Printf("Saved %s to secure storage", profileName)

		return nil
	},
}

var UpdateCredentialsCommand = cli.Command{
	Name:  "update",
	Usage: "Update existing credentials in secure storage",
	Action: func(c *cli.Context) error {
		profileName := c.Args().First()
		if profileName == "" {
			in := survey.Input{Message: "Profile Name: "}
			fmt.Println()
			err := testable.AskOne(&in, &profileName)
			if err != nil {
				return err
			}
		}

		secureIAMCredentialStorage := securestorage.NewSecureIAMCredentialStorage()
		_, err := secureIAMCredentialStorage.GetCredentials(profileName)
		if err != nil {
			return errors.Wrap(err, "error while looking up existing profile in secure storage")
		}
		var credentials aws.Credentials
		in1 := survey.Input{Message: "Access Key Id: "}
		fmt.Println()
		err = testable.AskOne(&in1, &credentials.AccessKeyID)
		if err != nil {
			return err
		}

		in2 := survey.Password{Message: "Secret Access Key: "}
		fmt.Println()
		err = testable.AskOne(&in2, &credentials.SecretAccessKey)
		if err != nil {
			return err
		}
		err = secureIAMCredentialStorage.StoreCredentials(profileName, credentials)
		if err != nil {
			return err
		}
		fmt.Printf("Updated %s in secure storage", profileName)

		return nil
	},
}

var ListCredentialsCommand = cli.Command{
	Name:  "list",
	Usage: "Lists the profiles in secure storage",
	Action: func(c *cli.Context) error {
		secureIAMCredentialStorage := securestorage.NewSecureIAMCredentialStorage()
		profiles, err := secureIAMCredentialStorage.SecureStorage.List()
		if err != nil {
			return err
		}
		for _, profile := range profiles {
			fmt.Printf("%s\n", profile.Key)
		}
		return nil
	},
}

var ClearCredentialsCommand = cli.Command{
	Name:  "clear",
	Usage: "Remove credentials from secure storage",
	Flags: []cli.Flag{
		&cli.BoolFlag{Name: "all", Aliases: []string{"a"}, Usage: "Remove all credentials from secure storage"},
	},
	Action: func(c *cli.Context) error {
		secureIAMCredentialStorage := securestorage.NewSecureIAMCredentialStorage()
		configPath := config.DefaultSharedConfigFilename()
		configFile, err := configparser.NewConfigParserFromFile(configPath)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if c.Bool("all") {
			profiles, err := secureIAMCredentialStorage.SecureStorage.List()
			if err != nil {
				return err
			}
			for _, profile := range profiles {
				fmt.Printf("Removing profile %s\n", profile.Key)
				secureIAMCredentialStorage.SecureStorage.Clear(profile.Key)
				sectionName := "profile " + profile.Key
				if configFile.HasSection(sectionName) {
					err = configFile.RemoveSection(sectionName)
					if err != nil {
						return err
					}
				}
			}
		} else {
			profileName := c.Args().First()
			if profileName == "" {
				in := survey.Input{Message: "Profile Name: "}
				fmt.Println()
				err := testable.AskOne(&in, &profileName)
				if err != nil {
					return err
				}
			}
			fmt.Printf("Removing profile %s\n", profileName)
			secureIAMCredentialStorage.SecureStorage.Clear(profileName)
			sectionName := "profile " + profileName
			if configFile.HasSection(sectionName) {
				err = configFile.RemoveSection(sectionName)
				if err != nil {
					return err
				}
			}
		}
		err = configFile.SaveWithDelimiter(configPath, "=")
		if err != nil {
			return err
		}
		fmt.Printf("Cleared credentials from secure storage")
		return nil
	},
}

package granted

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/core"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/bigkevmcd/go-configparser"

	"github.com/common-fate/granted/internal/build"
	"github.com/common-fate/granted/pkg/cfaws"
	"github.com/common-fate/granted/pkg/securestorage"
	"github.com/common-fate/granted/pkg/testable"
	"github.com/urfave/cli/v2"
)

var CredentialsCommand = cli.Command{
	Name:        "credentials",
	Usage:       "Manage secure IAM credentials",
	Subcommands: []*cli.Command{&AddCredentialsCommand, &ImportCredentialsCommand, &UpdateCredentialsCommand, &ListCredentialsCommand, &RemoveCredentialsCommand, &ExportCredentialsCommand},
}

var AddCredentialsCommand = cli.Command{
	Name:      "add",
	Usage:     "Add IAM credentials to secure storage",
	ArgsUsage: "[<profile>]",
	Action: func(c *cli.Context) error {
		profileName := c.Args().First()
		if profileName == "" {
			in := survey.Input{Message: "Profile Name:"}
			err := testable.AskOne(&in, &profileName, survey.WithValidator(survey.MinLength(1)))
			if err != nil {
				return err
			}
		}

		// validate the the profile does not already exist
		profiles, err := cfaws.LoadProfiles()
		if err != nil {
			return err
		}
		if profiles.HasProfile(profileName) {
			return fmt.Errorf("a profile with name %s already exists, you can import an existing profile using '%s credentials import %s", profileName, build.GrantedBinaryName(), profileName)
		}

		credentials, err := promptCredentials()
		if err != nil {
			return err
		}

		// store the credentials in secure storage
		secureIAMCredentialStorage := securestorage.NewSecureIAMCredentialStorage()
		err = secureIAMCredentialStorage.StoreCredentials(profileName, credentials)
		if err != nil {
			return err
		}
		err = updateOrCreateProfileWithCredentialProcess(profileName)
		if err != nil {
			return err
		}
		fmt.Printf("Saved %s to secure storage\n", profileName)

		return nil
	},
}

// addCredentialProcessToConfigfileProfile creates or updates a profile entry in the aws config file withs a granted credential_process entry
// this allows the profile to still work as expected with the AWS cli or other tools using the --profile flag
//
//	[profile my-profile]
//	credential_process = granted credential-process --profile=my-profile
func updateOrCreateProfileWithCredentialProcess(profileName string) error {
	configPath := config.DefaultSharedConfigFilename()
	configFile, err := configparser.NewConfigParserFromFile(configPath)
	if err != nil {
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
	return configFile.SaveWithDelimiter(configPath, "=")
}

func validateProfileForImport(ctx context.Context, profiles *cfaws.Profiles, profileName string, overwrite bool) error {
	secureIAMCredentialStorage := securestorage.NewSecureIAMCredentialStorage()
	if !profiles.HasProfile(profileName) {
		return fmt.Errorf("profile with name %s does not exist", profileName)
	}

	profile, err := profiles.LoadInitialisedProfile(ctx, profileName)
	if err != nil {
		return err
	}

	// ensure that the profile is an IAM profile
	if profile.ProfileType != "AWS_IAM" {
		return fmt.Errorf("profile %s does not have related IAM credentials", profileName)
	}
	// ensure that it is a root profile
	if len(profile.Parents) != 0 {
		return fmt.Errorf("profile %s uses a source_profile you should import the root profile '%s' instead. '%s credentials import %s'", profileName, profile.Parents[0].Name, build.GrantedBinaryName(), profile.Parents[0].Name)
	}
	// ensure it does not already exist in the secure storage
	existsInSecureStorage, err := secureIAMCredentialStorage.SecureStorage.HasKey(profileName)
	if err != nil {
		return err
	}
	if existsInSecureStorage && !overwrite {
		return fmt.Errorf("profile %s is already stored in secure storage.\nIf you were trying to update the credentials in secure storage, you can use '%s credentials update %s', or to overwrite the credentials in secure storage, run '%s credentials import --overwrite %s'", profileName, build.GrantedBinaryName(), profileName, build.GrantedBinaryName(), profileName)
	}
	if !profile.AWSConfig.Credentials.HasKeys() {
		return fmt.Errorf("profile %s does not have IAM credentials configured", profileName)
	}
	return nil
}

var ImportCredentialsCommand = cli.Command{
	Name:      "import",
	Usage:     "Import plaintext IAM user credentials from AWS credentials file into secure storage",
	ArgsUsage: "[<profile>]",
	Flags: []cli.Flag{
		&cli.BoolFlag{Name: "overwrite", Usage: "Overwrite an existing profile saved in secure storage with values from the AWS credentials file"},
	},
	Action: func(c *cli.Context) error {
		profileName := c.Args().First()
		profiles, err := cfaws.LoadProfiles()
		if err != nil {
			return err
		}

		if profileName == "" {
			in := survey.Select{Message: "Profile Name:", Options: profiles.ProfileNames}
			err := testable.AskOne(&in, &profileName, survey.WithValidator(func(ans interface{}) error {
				option := ans.(core.OptionAnswer)
				// Not all profiles are valid for importing, so ensure this profile is suitable, and inform the user if it is not + the reason
				return validateProfileForImport(c.Context, profiles, option.Value, c.Bool("overwrite"))
			}))
			if err != nil {
				return err
			}
		} else {
			err = validateProfileForImport(c.Context, profiles, profileName, c.Bool("overwrite"))
			if err != nil {
				return err
			}
		}

		profile, err := profiles.LoadInitialisedProfile(c.Context, profileName)
		if err != nil {
			return err
		}
		secureIAMCredentialStorage := securestorage.NewSecureIAMCredentialStorage()
		err = secureIAMCredentialStorage.StoreCredentials(profileName, profile.AWSConfig.Credentials)
		if err != nil {
			return err
		}
		// configure a profile in the config file with a credential process so the secure IAM credentials can be used
		err = updateOrCreateProfileWithCredentialProcess(profileName)
		if err != nil {
			return err
		}

		// remove the profile from the credentials file
		credentialsFilePath := config.DefaultSharedCredentialsFilename()
		credentialsFile, err := configparser.NewConfigParserFromFile(credentialsFilePath)
		if err != nil {
			return err
		}

		items, err := credentialsFile.Items(profileName)
		if err != nil {
			return err
		}

		configPath := config.DefaultSharedConfigFilename()
		configFile, err := configparser.NewConfigParserFromFile(configPath)
		if err != nil {
			return err
		}
		sectionName := "profile " + profileName

		// Merge options from the credentials profile to the config file profile.
		// if the same option is configured in the config file profile it takes precedence
		for k, v := range items {
			// omit sensitive values from the merge
			if !(k == "aws_access_key_id" || k == "aws_secret_access_key" || k == "aws_session_token") {
				has, err := configFile.HasOption(sectionName, k)
				if err != nil {
					return err
				}
				if !has {
					err = configFile.Set(sectionName, k, v)
					if err != nil {
						return err
					}
				}
			}
		}
		// save the updated config file after merging
		err = configFile.SaveWithDelimiter(configPath, "=")
		if err != nil {
			return err
		}

		// remove the plaintext profile from the credentials file
		err = credentialsFile.RemoveSection(profileName)
		if err != nil {
			return err
		}
		err = credentialsFile.SaveWithDelimiter(credentialsFilePath, "=")
		if err != nil {
			return err
		}
		fmt.Printf("Saved %s to secure storage\n", profileName)

		return nil
	},
}

func promptCredentials() (credentials aws.Credentials, err error) {
	in1 := survey.Input{Message: "Access Key ID:"}
	err = testable.AskOne(&in1, &credentials.AccessKeyID)
	if err != nil {
		return
	}
	in2 := survey.Password{Message: "Secret Sccess Key:"}
	err = testable.AskOne(&in2, &credentials.SecretAccessKey)
	if err != nil {
		return
	}
	return
}

var UpdateCredentialsCommand = cli.Command{
	Name:      "update",
	Usage:     "Update existing credentials in secure storage",
	ArgsUsage: "[<profile>]",
	Action: func(c *cli.Context) error {
		profileName := c.Args().First()
		secureIAMCredentialStorage := securestorage.NewSecureIAMCredentialStorage()

		if profileName == "" {
			profileNames, err := secureIAMCredentialStorage.SecureStorage.ListKeys()
			if err != nil {
				return err
			}
			if profileName == "" && len(profileNames) == 0 {
				fmt.Println("No credentials in secure storage")
				return nil
			}
			in := survey.Select{Message: "Profile Name:", Options: profileNames}
			err = testable.AskOne(&in, &profileName)
			if err != nil {
				return err
			}
		}

		has, err := secureIAMCredentialStorage.SecureStorage.HasKey(profileName)
		if err != nil {
			return err
		}
		if !has {
			return fmt.Errorf("no credentials exist for %s in secure storage. If you wanted to add a new profile, run '%s credentials add'", profileName, build.GrantedBinaryName())
		}
		credentials, err := promptCredentials()
		if err != nil {
			return err
		}
		err = secureIAMCredentialStorage.StoreCredentials(profileName, credentials)
		if err != nil {
			return err
		}
		fmt.Printf("Updated %s in secure storage\n", profileName)
		return nil
	},
}

var ListCredentialsCommand = cli.Command{
	Name:  "list",
	Usage: "Lists the profile names for credentials in secure storage",
	Action: func(c *cli.Context) error {
		secureIAMCredentialStorage := securestorage.NewSecureIAMCredentialStorage()
		profiles, err := secureIAMCredentialStorage.SecureStorage.List()
		if err != nil {
			return err
		}
		if len(profiles) == 0 {
			fmt.Fprintln(os.Stderr, "No IAM user credentials stored in secure storage")
			return nil
		}
		for _, profile := range profiles {
			fmt.Printf("%s\n", profile.Key)
		}
		return nil
	},
}

var RemoveCredentialsCommand = cli.Command{
	Name:      "remove",
	Usage:     "Remove credentials from secure storage and an associated profile if it exists in the AWS config file",
	ArgsUsage: "[<profile>]",
	Flags: []cli.Flag{
		&cli.BoolFlag{Name: "all", Aliases: []string{"a"}, Usage: "Remove all credentials from secure storage and an associated profile if it exists in the AWS config file"},
	},
	Action: func(c *cli.Context) error {
		secureIAMCredentialStorage := securestorage.NewSecureIAMCredentialStorage()
		configPath := config.DefaultSharedConfigFilename()
		configFile, err := configparser.NewConfigParserFromFile(configPath)
		if err != nil {
			return err
		}
		profileName := c.Args().First()
		secureProfileKeys, err := secureIAMCredentialStorage.SecureStorage.ListKeys()
		if err != nil {
			return err
		}
		var profileNames []string
		if c.Bool("all") {
			profileNames = append(profileNames, secureProfileKeys...)
		} else {
			if profileName == "" && len(secureProfileKeys) == 0 {
				fmt.Println("No credentials in secure storage")
				return nil
			}
			if profileName == "" {
				in := survey.Select{Message: "Profile Name:", Options: secureProfileKeys}
				err = testable.AskOne(&in, &profileName)
				if err != nil {
					return err
				}
			}
			profileNames = append(profileNames, profileName)
		}
		fmt.Printf(`Removing credentials from secure storage will cause them to be permanently deleted.
To avoid losing your credentials you may first want to export them to plaintext using 'granted credentials export-plaintext <profile name>'
This command will remove a profile with the same name from the AWS config file if it has a 'credential_process = granted credential-process --profile=<profile name>'
If you have already used 'granted credentials export-plaintext <profile name>' to export the credentials, the profile will not be removed by this command.

`)
		var confirm bool
		s := &survey.Confirm{
			Message: "Are you sure you want to remove these credentials and profile from your AWS config?",
			Default: true,
		}
		err = survey.AskOne(s, &confirm)
		if err != nil {
			return err
		}
		if !confirm {
			fmt.Printf("Cancelled clearing credentials\n")
			return nil
		}

		for _, profileName := range profileNames {
			fmt.Printf("Removing %s credentials from secure storage\n", profileName)
			err = secureIAMCredentialStorage.SecureStorage.Clear(profileName)
			if err != nil {
				return err
			}
			sectionName := "profile " + profileName
			if configFile.HasSection(sectionName) {
				hasCredentialProcess, err := configFile.HasOption(sectionName, "credential_process")
				if err != nil {
					return err
				}
				if hasCredentialProcess {
					credentialProcessOption, err := configFile.Get(sectionName, "credential_process")
					if err != nil {
						return err
					}
					if strings.HasPrefix(credentialProcessOption, fmt.Sprintf("%s credential-process", build.GrantedBinaryName())) {
						fmt.Printf("Removing profile %s AWS config file\n", profileName)
						err = configFile.RemoveSection(sectionName)
						if err != nil {
							return err
						}
					}
				}
			}
		}
		err = configFile.SaveWithDelimiter(configPath, "=")
		if err != nil {
			return err
		}
		fmt.Printf("Cleared credentials from secure storage\n")
		return nil
	},
}

var ExportCredentialsCommand = cli.Command{
	Name:      "export-plaintext",
	Usage:     "Export credentials from the secure storage to ~/.aws/credentials file in plaintext",
	ArgsUsage: "[<profile>]",
	Flags: []cli.Flag{
		&cli.BoolFlag{Name: "all", Aliases: []string{"a"}, Usage: "export all credentials from secure storage in plaintext"},
	},
	Action: func(c *cli.Context) error {
		secureIAMCredentialStorage := securestorage.NewSecureIAMCredentialStorage()
		profileName := c.Args().First()
		secureProfileKeys, err := secureIAMCredentialStorage.SecureStorage.ListKeys()
		if err != nil {
			return err
		}
		var profileNames []string
		if c.Bool("all") {
			profileNames = append(profileNames, secureProfileKeys...)
		} else {
			if profileName == "" && len(secureProfileKeys) == 0 {
				fmt.Println("No credentials in secure storage")
				return nil
			}

			if profileName == "" {
				in := survey.Select{Message: "Profile Name:", Options: secureProfileKeys}
				err = testable.AskOne(&in, &profileName)
				if err != nil {
					return err
				}
			}
			profileNames = append(profileNames, profileName)
		}

		for _, profileName := range profileNames {
			credentials, err := secureIAMCredentialStorage.GetCredentials(profileName)
			if err != nil {
				return err
			}
			//fetch parsed credentials file
			credentialsFilePath := config.DefaultSharedCredentialsFilename()
			credentialsFile, err := configparser.NewConfigParserFromFile(credentialsFilePath)
			if err != nil {
				return err
			}

			err = credentialsFile.AddSection(profileName)
			if err != nil {
				return err
			}

			if credentials.AccessKeyID != "" {
				if err := credentialsFile.Set(profileName, "aws_access_key_id", credentials.AccessKeyID); err != nil {
					return err
				}
			}
			if credentials.SecretAccessKey != "" {
				if err := credentialsFile.Set(profileName, "aws_secret_access_key", credentials.SecretAccessKey); err != nil {
					return err
				}
			}
			if credentials.SessionToken != "" {
				if err := credentialsFile.Set(profileName, "aws_session_token", credentials.SessionToken); err != nil {
					return err
				}
			}

			err = credentialsFile.SaveWithDelimiter(credentialsFilePath, "=")
			if err != nil {
				return err
			}
			configPath := config.DefaultSharedConfigFilename()
			configFile, err := configparser.NewConfigParserFromFile(configPath)
			if err != nil {
				return err
			}
			sectionName := "profile " + profileName
			if configFile.HasSection(sectionName) {
				has, err := configFile.HasOption(sectionName, "credential_process")
				if err != nil {
					return err
				}
				if has {
					items, err := configFile.Items(sectionName)
					if err != nil {
						return err
					}
					// if the result of removing the credential process is that the profile has not configuration, then just remove it completely.
					// the profile in the credential file will suffice
					// else just remove teh credential process line.
					// this avoids leaving the config file with an empty profile, which appears to be some kind of error when its not
					if len(items) > 1 {
						err = configFile.RemoveOption(sectionName, "credential_process")
						if err != nil {
							return err
						}
					} else {
						err = configFile.RemoveSection(sectionName)
						if err != nil {
							return err
						}
					}
					err = configFile.SaveWithDelimiter(configPath, "=")
					if err != nil {
						return err
					}

				}
			}

			fmt.Printf("Exported %s in plaintext from secure storage to %s\n", profileName, credentialsFilePath)
			fmt.Printf("The %s credentials have not been removed from secure storage. If you'd like to delete them, you can run '%s credentials remove %s'\n", profileName, build.GrantedBinaryName(), profileName)

		}
		return nil
	},
}

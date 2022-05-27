package cfaws

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/bigkevmcd/go-configparser"
	"github.com/fatih/color"
)

// ExportCredsToProfile will write assumed credentials to ~/.aws/credentials with a specified profile name header
func ExportCredsToProfile(profileName string, creds aws.Credentials) error {
	// fetch the parsed cred file
	credPath := config.DefaultSharedCredentialsFilename()

	//create it if it doesn't exist
	if _, err := os.Stat(credPath); os.IsNotExist(err) {

		f, err := os.Create(credPath)
		if err != nil {
			return err
		}
		err = f.Close()
		if err != nil {
			return err
		}
		fmt.Fprintln(color.Error, "Created file.")

	}

	credFile, err := configparser.NewConfigParserFromFile(credPath)
	if err != nil {
		return err
	}

	if credFile.HasSection(profileName) {
		//put the creds into options
		credFile.Set(profileName, "aws_access_key_id", creds.AccessKeyID)
		credFile.Set(profileName, "aws_secret_access_key", creds.SecretAccessKey)
		credFile.Set(profileName, "aws_session_token", creds.SessionToken)
		err = credFile.SaveWithDelimiter(credPath, "=")
		if err != nil {
			return err
		}

	} else {
		//create section and add creds
		err = credFile.AddSection(profileName)
		if err != nil {
			return err
		}

		credFile.Set(profileName, "aws_access_key_id", creds.AccessKeyID)
		credFile.Set(profileName, "aws_secret_access_key", creds.SecretAccessKey)
		credFile.Set(profileName, "aws_session_token", creds.SessionToken)

		err = credFile.SaveWithDelimiter(credPath, "=")
		if err != nil {
			return err
		}
	}

	return nil
}

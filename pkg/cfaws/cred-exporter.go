package cfaws

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/bigkevmcd/go-configparser"
	gconfig "github.com/common-fate/granted/pkg/config"

	"github.com/fatih/color"
)

func WriteProfileToDefaultCredentialsFile(profileName string, creds aws.Credentials) error {
	return WriteProfileToCredentialsFile(profileName, creds, config.DefaultSharedCredentialsFilename())
}

// WriteProfileToCredentialsFile will write assumed credentials to the credentials file at the specified path with a specified profile name header
func WriteProfileToCredentialsFile(profileName string, creds aws.Credentials, path string) error {
	//create it if it doesn't exist
	if _, err := os.Stat(path); os.IsNotExist(err) {

		f, err := os.Create(path)
		if err != nil {
			return err
		}
		err = f.Close()
		if err != nil {
			return err
		}
		fmt.Fprintln(color.Error, "Created file.")

	}

	credFile, err := configparser.NewConfigParserFromFile(path)
	if err != nil {
		return err
	}

	cfg, err := gconfig.Load()
	if err != nil {
		return err
	}

	if cfg.ExportCredentialSuffix != "" {
		profileName = profileName + "-" + cfg.ExportCredentialSuffix
	}

	if credFile.HasSection(profileName) {
		err := credFile.RemoveSection(profileName)
		if err != nil {
			return err
		}
	}
	err = credFile.AddSection(profileName)
	if err != nil {
		return err
	}
	//put the creds into options
	err = credFile.Set(profileName, "aws_access_key_id", creds.AccessKeyID)
	if err != nil {
		return err
	}
	err = credFile.Set(profileName, "aws_secret_access_key", creds.SecretAccessKey)
	if err != nil {
		return err
	}
	err = credFile.Set(profileName, "aws_session_token", creds.SessionToken)
	if err != nil {
		return err
	}
	err = credFile.SaveWithDelimiter(path, "=")
	if err != nil {
		return err
	}

	return nil
}

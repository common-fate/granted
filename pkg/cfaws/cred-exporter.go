package cfaws

import (
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/bigkevmcd/go-configparser"
	"github.com/common-fate/clio"
	gconfig "github.com/common-fate/granted/pkg/config"
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
		clio.Info("An AWS credentials file was not found at %s so it has been created", credPath)

	}

	credFile, err := configparser.NewConfigParserFromFile(credPath)
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
	err = credFile.SaveWithDelimiter(credPath, "=")
	if err != nil {
		return err
	}

	return nil
}

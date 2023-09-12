package cfaws

import (
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/common-fate/clio"
	gconfig "github.com/common-fate/granted/pkg/config"
	"gopkg.in/ini.v1"
)

// ExportCredsToProfile will write assumed credentials to ~/.aws/credentials with a specified profile name header
func ExportCredsToProfile(profileName string, creds aws.Credentials) error {
	// fetch the parsed cred file
	credPath := GetAWSCredentialsPath()

	// create it if it doesn't exist
	if _, err := os.Stat(credPath); os.IsNotExist(err) {

		f, err := os.Create(credPath)
		if err != nil {
			return err
		}
		err = f.Close()
		if err != nil {
			return err
		}
		clio.Infof("An AWS credentials file was not found at %s so it has been created", credPath)

	}

	credentialsFile, err := ini.LoadSources(ini.LoadOptions{
		AllowNonUniqueSections:  false,
		SkipUnrecognizableLines: false,
	}, credPath)
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

	credentialsFile.DeleteSection(profileName)
	section, err := credentialsFile.NewSection(profileName)
	if err != nil {
		return err
	}
	// put the creds into options
	err = section.ReflectFrom(&struct {
		AWSAccessKeyID     string `ini:"aws_access_key_id"`
		AWSSecretAccessKey string `ini:"aws_secret_access_key"`
		AWSSessionToken    string `ini:"aws_session_token,omitempty"`
	}{
		AWSAccessKeyID:     creds.AccessKeyID,
		AWSSecretAccessKey: creds.SecretAccessKey,
		AWSSessionToken:    creds.SessionToken,
	})
	if err != nil {
		return err
	}
	return credentialsFile.SaveTo(credPath)
}

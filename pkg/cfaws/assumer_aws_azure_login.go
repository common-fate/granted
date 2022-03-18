package cfaws

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/bigkevmcd/go-configparser"
	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
)

// Implements Assumer
type AwsAzureLoginAssumer struct {
}

//https://github.com/sportradar/aws-azure-login

// then fetch them from the environment for use
func (aal *AwsAzureLoginAssumer) AssumeTerminal(c *cli.Context, cfg *CFSharedConfig, args []string) (aws.Credentials, error) {
	//check to see if the creds are already exported
	creds, err := GetCredentialsCreds(c.Context, cfg)

	if err == nil {
		return creds, nil
	}

	//request for the creds if they are invalid
	a := []string{fmt.Sprintf("--profile=%s", cfg.AWSConfig.Profile)}
	a = append(a, args...)

	cmd := exec.Command("aws-azure-login", a...)

	cmd.Stdout = color.Error
	cmd.Stdin = os.Stdin
	cmd.Stderr = color.Error
	err = cmd.Run()
	if err != nil {
		return aws.Credentials{}, err
	}
	// reload the profile from disk to check for the new credentials
	awsCfg, err := config.LoadDefaultConfig(c.Context,
		config.WithSharedConfigProfile(cfg.AWSConfig.Profile),
	)
	if err != nil {
		return aws.Credentials{}, err
	}
	return aws.NewCredentialsCache(awsCfg.Credentials).Retrieve(c.Context)
}

func (aal *AwsAzureLoginAssumer) AssumeConsole(c *cli.Context, cfg *CFSharedConfig, args []string) (aws.Credentials, error) {
	return aal.AssumeTerminal(c, cfg, args)
}

// A unique key which identifies this assumer e.g AWS-SSO or GOOGLE-AWS-AUTH
func (aal *AwsAzureLoginAssumer) Type() string {
	return "AWS_AZURE_LOGIN"
}

// inspect for any items on the profile prefixed with "AZURE_"
func (aal *AwsAzureLoginAssumer) ProfileMatchesType(rawProfile configparser.Dict, parsedProfile config.SharedConfig) bool {
	for k := range rawProfile {
		if strings.HasPrefix(k, "azure_") {
			return true
		}
	}
	return false
}

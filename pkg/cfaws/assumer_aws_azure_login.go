package cfaws

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/bigkevmcd/go-configparser"
	"github.com/common-fate/granted/pkg/browsers"
	"github.com/fatih/color"
)

// Implements Assumer
type AwsAzureLoginAssumer struct {
}

//https://github.com/sportradar/aws-azure-login

// then fetch them from the environment for use
func (aal *AwsAzureLoginAssumer) AssumeTerminal(ctx context.Context, c *CFSharedConfig, configOpts ConfigOpts) (aws.Credentials, error) {
	//check to see if the creds are already exported
	creds, err := GetCredentialsCreds(ctx, c)

	if err == nil {
		return creds, nil
	}

	//request for the creds if they are invalid
	a := []string{fmt.Sprintf("--profile=%s", c.Name)}
	a = append(a, configOpts.Args...)

	cmd := exec.Command("aws-azure-login", a...)

	cmd.Stdout = color.Error
	cmd.Stdin = os.Stdin
	cmd.Stderr = color.Error
	err = cmd.Run()
	if err != nil {
		return aws.Credentials{}, err
	}
	// reload the profile from disk to check for the new credentials
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithSharedConfigProfile(c.Name),
	)
	if err != nil {
		return aws.Credentials{}, err
	}
	return aws.NewCredentialsCache(cfg.Credentials).Retrieve(ctx)
}

func (aal *AwsAzureLoginAssumer) AssumeConsole(ctx context.Context, c *CFSharedConfig, browserOpts browsers.BrowserOpts, configOpts ConfigOpts) (aws.Credentials, error) {
	return aal.AssumeTerminal(ctx, c, configOpts)
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

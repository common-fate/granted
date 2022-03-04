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
)

// Implements Assumer
type AwsAzureLoginAssumer struct {
}

//https://github.com/sportradar/aws-azure-login

// launch the aws-google-auth utility to generate the credentials
// then fetch them from the environment for use
func (aal *AwsAzureLoginAssumer) AssumeTerminal(ctx context.Context, c *CFSharedConfig, args []string) (aws.Credentials, error) {
	cfg, err := c.AwsConfig(ctx, false)
	if err != nil {
		return aws.Credentials{}, err
	}
	creds, _ := aws.NewCredentialsCache(cfg.Credentials).Retrieve(ctx)

	if creds.HasKeys() && !creds.Expired() {
		return creds, nil
	}

	a := []string{fmt.Sprintf("--profile=%s", c.Name)}
	a = append(a, args...)

	cmd := exec.Command("aws-azure-login", a...)

	cmd.Stdout = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return aws.Credentials{}, err
	}
	cfg, err = c.AwsConfig(ctx, false)
	if err != nil {
		return aws.Credentials{}, err
	}
	creds, err = aws.NewCredentialsCache(cfg.Credentials).Retrieve(ctx)
	if err != nil {
		return aws.Credentials{}, err
	}
	return creds, nil
}

func (aal *AwsAzureLoginAssumer) AssumeConsole(ctx context.Context, c *CFSharedConfig, args []string) (aws.Credentials, error) {
	return aal.AssumeTerminal(ctx, c, args)
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

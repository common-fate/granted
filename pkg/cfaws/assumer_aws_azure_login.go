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

// launch the aws-google-auth utility to generate the credentials
// then fetch them from the environment for use
func (aal *AwsAzureLoginAssumer) AssumeTerminal(ctx context.Context, c *CFSharedConfig) (aws.Credentials, error) {
	cmd := exec.Command("aws-azure-login", fmt.Sprintf("--profile=%s", c.Name))

	cmd.Stdout = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return aws.Credentials{}, err
	}
	creds := GetEnvCredentials(ctx)
	if !creds.HasKeys() {
		return aws.Credentials{}, fmt.Errorf("no credentials exported to terminal when using %s to assume profile: %s", aal.Type(), c.Name)
	}
	return creds, nil
}

func (aal *AwsAzureLoginAssumer) AssumeConsole(ctx context.Context, c *CFSharedConfig) (aws.Credentials, error) {
	return aal.AssumeTerminal(ctx, c)
}

// A unique key which identifies this assumer e.g AWS-SSO or GOOGLE-AWS-AUTH
func (aal *AwsAzureLoginAssumer) Type() string {
	return "AWS_AZURE_LOGIN"
}

// inspect for any items on the profile prefixed with "google_config."
func (aal *AwsAzureLoginAssumer) ProfileMatchesType(rawProfile configparser.Dict, parsedProfile config.SharedConfig) bool {
	for k := range rawProfile {
		if strings.HasPrefix(k, "azure_") {
			return true
		}
	}
	return false
}

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
type AwsGoogleAuthAssumer struct {
}

// launch the aws-google-auth utility to generate the credentials
// then fetch them from the environment for use
func (aia *AwsGoogleAuthAssumer) AssumeTerminal(ctx context.Context, c *Profile, configOpts ConfigOpts) (aws.Credentials, error) {
	cmd := exec.Command("aws-google-auth", fmt.Sprintf("--profile=%s", c.Name))

	cmd.Stdout = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return aws.Credentials{}, err
	}
	creds := GetEnvCredentials(ctx)
	if !creds.HasKeys() {
		return aws.Credentials{}, fmt.Errorf("no credentials exported to terminal when using %s to assume profile: %s", aia.Type(), c.Name)
	}
	return creds, nil
}

func (aia *AwsGoogleAuthAssumer) AssumeConsole(ctx context.Context, c *Profile, configOpts ConfigOpts) (aws.Credentials, error) {
	return aia.AssumeTerminal(ctx, c, configOpts)
}

// A unique key which identifies this assumer e.g AWS-SSO or GOOGLE-AWS-AUTH
func (aia *AwsGoogleAuthAssumer) Type() string {
	return "AWS_GOOGLE_AUTH"
}

// inspect for any items on the profile prefixed with "google_config."
func (aia *AwsGoogleAuthAssumer) ProfileMatchesType(rawProfile configparser.Dict, parsedProfile config.SharedConfig) bool {
	for k := range rawProfile {
		if strings.HasPrefix(k, "google_config.") {
			return true
		}
	}
	return false
}

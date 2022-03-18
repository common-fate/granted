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
type AwsGoogleAuthAssumer struct {
}

// launch the aws-google-auth utility to generate the credentials
// then fetch them from the environment for use
func (aia *AwsGoogleAuthAssumer) AssumeTerminal(c *cli.Context, cfg *CFSharedConfig, args []string) (aws.Credentials, error) {
	cmd := exec.Command("aws-google-auth", fmt.Sprintf("--profile=%s", cfg.AWSConfig.Profile))

	cmd.Stdout = color.Error
	cmd.Stdin = os.Stdin
	cmd.Stderr = color.Error
	err := cmd.Run()
	if err != nil {
		return aws.Credentials{}, err
	}
	creds := GetEnvCredentials(c.Context)
	if !creds.HasKeys() {
		return aws.Credentials{}, fmt.Errorf("no credentials exported to terminal when using %s to assume profile: %s", aia.Type(), cfg.DisplayName)
	}
	return creds, nil
}

func (aia *AwsGoogleAuthAssumer) AssumeConsole(c *cli.Context, cfg *CFSharedConfig, args []string) (aws.Credentials, error) {
	return aia.AssumeTerminal(c, cfg, args)
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

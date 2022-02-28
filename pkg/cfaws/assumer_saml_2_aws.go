package cfaws

import (
	"context"
	"os"
	"os/exec"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/bigkevmcd/go-configparser"
)

// Implements Assumer
type Saml2AwsAssumer struct {
}

// launch the saml2aws utility to generate the credentials
// then fetch them from the environment for use
func (s2a *Saml2AwsAssumer) AssumeTerminal(ctx context.Context, c *CFSharedConfig) (aws.Credentials, error) {
	var args []string
	for k, v := range c.RawConfig {
		if k == "credential_process" && strings.HasPrefix(v, "saml2aws") {
			args = strings.Split(strings.TrimPrefix(v, "saml2aws"), " ")
			break
		}
	}

	// https://github.com/Versent/saml2aws#using-saml2aws-as-credential-process
	// attempt to run the credential process for this profile
	cmd := exec.Command("saml2aws", args...)
	cmd.Stdout = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return aws.Credentials{}, err
	}

	// expect that this process has written the credentials to a profile in the ~/.aws/credentials file
	// we can now use the standard AWS sdk to get credentials for the profile
	cfg, err := c.AwsConfig(ctx, false)
	if err != nil {
		return aws.Credentials{}, err
	}
	creds, err := aws.NewCredentialsCache(cfg.Credentials).Retrieve(ctx)
	if err != nil {
		return aws.Credentials{}, err
	}
	return creds, nil
}

func (s2a *Saml2AwsAssumer) AssumeConsole(ctx context.Context, c *CFSharedConfig) (aws.Credentials, error) {
	return s2a.AssumeTerminal(ctx, c)
}

// A unique key which identifies this assumer e.g AWS-SSO or GOOGLE-AWS-AUTH
func (s2a *Saml2AwsAssumer) Type() string {
	return "SAML_2_AWS"
}

// inspect for any credential processes with the saml2aws tool
func (s2a *Saml2AwsAssumer) ProfileMatchesType(rawProfile configparser.Dict, parsedProfile config.SharedConfig) bool {
	for k, v := range rawProfile {
		if k == "credential_process" && strings.HasPrefix(v, "saml2aws") {
			return true
		}
	}
	return false
}

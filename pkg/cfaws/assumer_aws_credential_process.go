package cfaws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/processcreds"
	"github.com/bigkevmcd/go-configparser"
)

// Implements Assumer using the aws credential_process standard
type CredentialProcessAssumer struct {
}

func (cpa *CredentialProcessAssumer) AssumeTerminal(ctx context.Context, c *CFSharedConfig, configOpts ConfigOpts) (aws.Credentials, error) {
	var credProcessCommand string
	for k, v := range c.RawConfig {
		if k == "credential_process" {
			credProcessCommand = v
			break
		}
	}
	p := processcreds.NewProvider(credProcessCommand)
	return p.Retrieve(ctx)

}

func (cpa *CredentialProcessAssumer) AssumeConsole(ctx context.Context, c *CFSharedConfig, configOpts ConfigOpts) (aws.Credentials, error) {
	return cpa.AssumeTerminal(ctx, c, configOpts)
}

// A unique key which identifies this assumer e.g AWS-SSO or GOOGLE-AWS-AUTH
func (cpa *CredentialProcessAssumer) Type() string {
	return "AWS_CREDENTIAL_PROCESS"
}

// inspect for any credential processes with the saml2aws tool
func (cpa *CredentialProcessAssumer) ProfileMatchesType(rawProfile configparser.Dict, parsedProfile config.SharedConfig) bool {
	for k := range rawProfile {
		if k == "credential_process" {
			return true
		}
	}
	return false
}

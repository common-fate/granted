package cfaws

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/bigkevmcd/go-configparser"
)

// Implements Assumer
type AwsGoogleAuthAssumer struct {
}

// Default behaviour is to use the sdk to retrieve the credentials from the file
// For launching the console there is an extra step GetFederationToken that happens after this to get a session token
func (aia *AwsGoogleAuthAssumer) AssumeTerminal(ctx context.Context, c *CFSharedConfig) (aws.Credentials, error) {

	cmd := exec.Command(fmt.Sprintf("aws-google-auth  --profile=%s", c.Name))
	err := cmd.Run()
	if err != nil {
		return aws.Credentials{}, err
	}

}

// if required will get a FederationToken to be used to launch the console
// This is required is the iam profile does not assume a role using sts.AssumeRole
func (aia *AwsGoogleAuthAssumer) AssumeConsole(ctx context.Context, c *CFSharedConfig) (aws.Credentials, error) {
	if c.AWSConfig.RoleARN == "" {
		return getFederationToken(ctx, c)
	} else {
		// profile assume a role
		return aia.AssumeTerminal(ctx, c)
	}
}

// A unique key which identifies this assumer e.g AWS-SSO or GOOGLE-AWS-AUTH
func (aia *AwsGoogleAuthAssumer) Type() string {
	return "AWS_IAM"
}

// Matches the profile type on whether it is not an sso profile.
// this will also match other types that are not sso profiles so it should be the last option checked when determining the profile type
func (aia *AwsGoogleAuthAssumer) ProfileMatchesType(rawProfile configparser.Dict, parsedProfile config.SharedConfig) bool {
	return parsedProfile.SSOAccountID == ""
}

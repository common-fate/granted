package cfaws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/bigkevmcd/go-configparser"
)

// Implements Assumer
type AwsIamAssumer struct {
}

// Default behaviour is to use the sdk to retrieve the credentials from the file
// For launching the console there is an extra step GetFederationToken that happens after this to get a session token
func (aia *AwsIamAssumer) AssumeTerminal(ctx context.Context, c *CFSharedConfig) (aws.Credentials, error) {
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

// if required will get a FederationToken to be used to launch the console
// This is required is the iam profile does not assume a role using sts.AssumeRole
func (aia *AwsIamAssumer) AssumeConsole(ctx context.Context, c *CFSharedConfig) (aws.Credentials, error) {
	if c.AWSConfig.RoleARN == "" {
		return getFederationToken(ctx, c)
	} else {
		// profile assume a role
		return aia.AssumeTerminal(ctx, c)
	}
}

// A unique key which identifies this assumer e.g AWS-SSO or GOOGLE-AWS-AUTH
func (aia *AwsIamAssumer) Type() string {
	return "AWS_IAM"
}

// Matches the profile type on whether it is not an sso profile.
// this will also match other types that are not sso profiles so it should be the last option checked when determining the profile type
func (aia *AwsIamAssumer) ProfileMatchesType(rawProfile configparser.Dict, parsedProfile config.SharedConfig) bool {
	return parsedProfile.SSOAccountID == ""
}

// GetFederationToken is used when launching a console session with longlived IAM credentials profiles
func getFederationToken(ctx context.Context, c *CFSharedConfig) (aws.Credentials, error) {
	cfg, err := c.AwsConfig(ctx, false)
	if err != nil {
		return aws.Credentials{}, err
	}
	client := sts.NewFromConfig(cfg)
	out, err := client.GetFederationToken(ctx, &sts.GetFederationTokenInput{Name: aws.String("Granted@" + c.Name)})
	if err != nil {
		return aws.Credentials{}, err
	}
	return TypeCredsToAwsCreds(*out.Credentials), err

}

package cfaws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/bigkevmcd/go-configparser"
	"github.com/urfave/cli/v2"
)

// Implements Assumer
type AwsIamAssumer struct {
}

// Default behaviour is to use the sdk to retrieve the credentials from the file
// For launching the console there is an extra step GetFederationToken that happens after this to get a session token
func (aia *AwsIamAssumer) AssumeTerminal(c *cli.Context, cfg *CFSharedConfig, args []string) (aws.Credentials, error) {

	opts := []func(*config.LoadOptions) error{
		// load the config profile
		config.WithSharedConfigProfile(cfg.Name),
	}

	//load the creds from the credentials file
	AwsCfg, err := config.LoadDefaultConfig(c.Context, opts...)
	if err != nil {
		return aws.Credentials{}, err
	}

	creds, err := aws.NewCredentialsCache(AwsCfg.Credentials).Retrieve(c.Context)
	if err != nil {
		return aws.Credentials{}, err
	}
	return creds, nil
}

// if required will get a FederationToken to be used to launch the console
// This is required is the iam profile does not assume a role using sts.AssumeRole
func (aia *AwsIamAssumer) AssumeConsole(c *cli.Context, cfg *CFSharedConfig, args []string) (aws.Credentials, error) {
	if cfg.AWSConfig.RoleARN == "" {
		return getFederationToken(c.Context, cfg)
	} else {
		// profile assume a role
		return aia.AssumeTerminal(c, cfg, args)
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
	cfg := aws.NewConfig()
	r, _, err := c.Region(ctx)
	if err != nil {
		return aws.Credentials{}, err
	}
	cfg.Region = r
	client := sts.NewFromConfig(*cfg)
	out, err := client.GetFederationToken(ctx, &sts.GetFederationTokenInput{Name: aws.String("Granted@" + c.Name)})
	if err != nil {
		return aws.Credentials{}, err
	}
	return TypeCredsToAwsCreds(*out.Credentials), err

}

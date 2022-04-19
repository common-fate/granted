package cfaws

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/bigkevmcd/go-configparser"
)

// Implements Assumer
type AwsIamAssumer struct {
}

// Default behaviour is to use the sdk to retrieve the credentials from the file
// For launching the console there is an extra step GetFederationToken that happens after this to get a session token
func (aia *AwsIamAssumer) AssumeTerminal(ctx context.Context, c *CFSharedConfig, configOpts ConfigOpts) (aws.Credentials, error) {

	duration := time.Hour

	if configOpts.Duration != 0 {
		duration = configOpts.Duration
	}

	opts := []func(*config.LoadOptions) error{
		// load the config profile
		config.WithSharedConfigProfile(c.Name),
		config.WithAssumeRoleCredentialOptions(func(aro *stscreds.AssumeRoleOptions) {
			// set the token provider up
			aro.TokenProvider = MfaTokenProvider
			aro.Duration = duration

			// If the mfa_serial is defined on the root profile, we need to set it in this config so that the aws SDK knows to prompt for MFA token
			if len(c.Parents) > 0 {
				if c.Parents[0].AWSConfig.MFASerial != "" {
					aro.SerialNumber = aws.String(c.Parents[0].AWSConfig.MFASerial)
				}
			}
		}),
	}

	//load the creds from the credentials file
	cfg, err := config.LoadDefaultConfig(ctx, opts...)
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
func (aia *AwsIamAssumer) AssumeConsole(ctx context.Context, c *CFSharedConfig, configOpts ConfigOpts) (aws.Credentials, error) {
	if c.AWSConfig.RoleARN == "" {
		return getFederationToken(ctx, c)
	} else {
		// profile assume a role
		return aia.AssumeTerminal(ctx, c, configOpts)
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

var allowAllPolicy = `{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "AllowAll",
            "Effect": "Allow",
            "Action": "*",
            "Resource": "*"
        }
    ]
}`

// GetFederationToken is used when launching a console session with longlived IAM credentials profiles
// GetFederation token uses an allow all IAM policy so that the console session will be able to access everything
// If this is not provided, the session cannot do anything in the console
func getFederationToken(ctx context.Context, c *CFSharedConfig) (aws.Credentials, error) {
	opts := []func(*config.LoadOptions) error{
		// load the config profile
		config.WithSharedConfigProfile(c.Name),
	}

	//load the creds from the credentials file
	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return aws.Credentials{}, err
	}

	client := sts.NewFromConfig(cfg)
	name := "Granted@" + c.Name

	if len(name) > 32 {
		name = name[0:32]
	}
	out, err := client.GetFederationToken(ctx, &sts.GetFederationTokenInput{Name: aws.String(name)})

	if err != nil {
		return aws.Credentials{}, err
	}
	return TypeCredsToAwsCreds(*out.Credentials), err

}

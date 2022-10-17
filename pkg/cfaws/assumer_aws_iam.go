package cfaws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/bigkevmcd/go-configparser"
	"github.com/common-fate/granted/pkg/securestorage"
	"github.com/fatih/color"
)

// Implements Assumer
type AwsIamAssumer struct {
}

// Default behaviour is to use the sdk to retrieve the credentials from the file
// For launching the console there is an extra step GetFederationToken that happens after this to get a session token
func (aia *AwsIamAssumer) AssumeTerminal(ctx context.Context, c *Profile, configOpts ConfigOpts) (aws.Credentials, error) {
	if c.HasSecureStorageIAMCredentials {
		secureIAMCredentialStorage := securestorage.NewSecureIAMCredentialStorage()
		return secureIAMCredentialStorage.GetCredentials(c.Name)
	}

	//using ~/.aws/credentials file for creds
	opts := []func(*config.LoadOptions) error{
		// load the config profile
		config.WithSharedConfigProfile(c.Name),
		config.WithAssumeRoleCredentialOptions(func(aro *stscreds.AssumeRoleOptions) {
			// set the token provider up
			aro.TokenProvider = MfaTokenProvider
			aro.Duration = configOpts.Duration

			// If the mfa_serial is defined on the root profile, we need to set it in this config so that the aws SDK knows to prompt for MFA token
			if len(c.Parents) > 0 {
				if c.Parents[0].AWSConfig.MFASerial != "" {
					aro.SerialNumber = aws.String(c.Parents[0].AWSConfig.MFASerial)

				}
			}
			if c.AWSConfig.RoleSessionName != "" {
				aro.RoleSessionName = c.AWSConfig.RoleSessionName
			} else {
				aro.RoleSessionName = sessionName()
			}
		}),
	}

	// load the creds from the credentials file
	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return aws.Credentials{}, err
	}

	credentials, err := aws.NewCredentialsCache(cfg.Credentials).Retrieve(ctx)
	if err != nil {
		return aws.Credentials{}, err
	}

	//inform the user about using the secure storage to securely store IAM user credentials
	// if it has no parents and it reached this point, it must have had plain text credentials
	// if it has parents, and the root is not a secure storage iam profile, then it has plain text credentials
	if len(c.Parents) == 0 || !c.Parents[0].HasSecureStorageIAMCredentials {
		fmt.Fprintf(color.Error, "Profile %s has plaintext credentials stored in the AWS credentials file.\n", c.Name)
		fmt.Fprintf(color.Error, "To move the credentials to secure storage, run 'granted credentials import %s'.\n", c.Name)
	}

	return credentials, nil

}

// if required will get a FederationToken to be used to launch the console
// This is required is the iam profile does not assume a role using sts.AssumeRole
func (aia *AwsIamAssumer) AssumeConsole(ctx context.Context, c *Profile, configOpts ConfigOpts) (aws.Credentials, error) {
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
func getFederationToken(ctx context.Context, c *Profile) (aws.Credentials, error) {
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

	out, err := client.GetFederationToken(ctx, &sts.GetFederationTokenInput{Name: aws.String(sessionName()), Policy: aws.String(allowAllPolicy)})

	if err != nil {
		return aws.Credentials{}, err
	}
	return TypeCredsToAwsCreds(*out.Credentials), err

}

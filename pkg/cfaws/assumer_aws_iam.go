package cfaws

import (
	"context"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/aws-sdk-go-v2/service/sts/types"
	"github.com/common-fate/clio"
	"github.com/common-fate/granted/pkg/securestorage"
	"gopkg.in/ini.v1"
)

// Implements Assumer
type AwsIamAssumer struct {
}

// Default behaviour is to use the sdk to retrieve the credentials from the file
// For launching the console there is an extra step GetFederationToken that happens after this to get a session token
func (aia *AwsIamAssumer) AssumeTerminal(ctx context.Context, c *Profile, configOpts ConfigOpts) (aws.Credentials, error) {
	// check if the valid credentials are available in session credential store

	sessionCredStorage := securestorage.NewSecureSessionCredentialStorage()

	cachedCreds, err := sessionCredStorage.GetCredentials(c.AWSConfig.Profile)
	if err != nil {
		clio.Debugw("error loading cached credentials", "error", err)
	} else if cachedCreds != nil && !cachedCreds.Expired() {
		clio.Debugw("credentials found in cache", "expires", cachedCreds.Expires.String(), "canExpire", cachedCreds.CanExpire, "timeNow", time.Now().String())
		return *cachedCreds, err
	}

	clio.Debugw("refreshing credentials", "reason", "not found")

	if c.HasSecureStorageIAMCredentials {
		secureIAMCredentialStorage := securestorage.NewSecureIAMCredentialStorage()
		creds, err := secureIAMCredentialStorage.GetCredentials(c.Name)
		if err != nil {
			return aws.Credentials{}, err
		}
		/**If the IAM credentials in secure storage are valid and no MFA is required:
		*[profile example]
		*region             = us-west-2
		*credential_process = dgranted credential-process --profile=example
		**/
		if c.AWSConfig.MFASerial == "" {
			return creds, nil
		}

		cfg, err := config.LoadDefaultConfig(ctx, config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(creds.AccessKeyID, creds.SecretAccessKey, creds.SessionToken)))
		if err != nil {
			return aws.Credentials{}, err
		}
		/**If the IAM credentials in secure storage MFA is required:
		*[profile example]
		*region             = us-west-2
		*mfa_serial         = arn:aws:iam::616777145260:mfa
		*credential_process = dgranted credential-process --profile=example
		**/
		clio.Debugw("generating temporary credentials", "assumer", aia.Type(), "profile_type", "base_profile_with_mfa")
		creds, err = aia.getTemporaryCreds(ctx, cfg, c, configOpts)
		if err != nil {
			return aws.Credentials{}, err
		}

		if err := sessionCredStorage.StoreCredentials(c.AWSConfig.Profile, creds); err != nil {
			clio.Warnf("Error caching credentials, MFA token will be requested before current token is expired")
		}

		return creds, nil
	}

	//using ~/.aws/credentials file for creds
	opts := []func(*config.LoadOptions) error{
		// load the config profile
		config.WithSharedConfigProfile(c.Name),
	}

	var credentials aws.Credentials
	// if the aws profile contains 'role_arn' then having this option will return the temporary credentials
	if c.AWSConfig.RoleARN != "" {
		clio.Debugw("generating temporary credentials", "assumer", aia.Type(), "profile_type", "with_role_arn")
		opts = append(opts, config.WithAssumeRoleCredentialOptions(func(aro *stscreds.AssumeRoleOptions) {
			// check if the MFAToken code is provided as argument
			// if provided then use it instead of prompting for MFAToken code.
			if configOpts.MFATokenCode != "" {
				aro.TokenProvider = func() (string, error) {
					return configOpts.MFATokenCode, nil
				}
			} else {
				// set the token provider up
				aro.TokenProvider = MfaTokenProvider
			}
			aro.Duration = configOpts.Duration

			/**If the mfa_serial is defined on the root profile, we need to set it in this config so that the aws SDK knows to prompt for MFA token:
			*[profile base]
			*region             = us-west-2
			*mfa_serial         = arn:aws:iam::616777145260:mfa

			*[profile prod]
			*role_arn       = XXXXXXX
			*source_profile = base
			**/
			if len(c.Parents) > 0 {
				if c.Parents[0].AWSConfig.MFASerial != "" {
					aro.SerialNumber = aws.String(c.Parents[0].AWSConfig.MFASerial)

				}
			} else {
				if c.AWSConfig.MFASerial != "" {
					aro.SerialNumber = aws.String(c.AWSConfig.MFASerial)
				}
			}
			if c.AWSConfig.RoleSessionName != "" {
				aro.RoleSessionName = c.AWSConfig.RoleSessionName
			} else {
				aro.RoleSessionName = sessionName()
			}
		}))

		cfg, err := config.LoadDefaultConfig(ctx, opts...)
		if err != nil {
			return aws.Credentials{}, err
		}

		credentials, err = aws.NewCredentialsCache(cfg.Credentials).Retrieve(ctx)
		if err != nil {
			return aws.Credentials{}, err
		}

	} else {
		// load the creds from the credentials file
		cfg, err := config.LoadDefaultConfig(ctx, opts...)
		if err != nil {
			return aws.Credentials{}, err
		}

		/**
		* Retrieve STS credentials when a base profile uses MFA
		*
		* ~/.aws/config
		* [profile prod]
		* region = ***
		* mfa_serial = ***
		*
		* ~/.aws/credentials
		* [profile prod]
		* aws_access_key_id = ***
		* aws_secret_access_key = ***
		**/
		if c.AWSConfig.MFASerial != "" {
			clio.Debugw("generating temporary credentials", "assumer", aia.Type(), "profile_type", "base_profile_with_mfa")
			credentials, err = aia.getTemporaryCreds(ctx, cfg, c, configOpts)
			if err != nil {
				return aws.Credentials{}, err
			}
		} else {
			// else for normal shared credentails, retrieve the long living credentials
			clio.Debugw("generating long-lived credentials", "assumer", aia.Type(), "profile_type", "credentials")
			credentials, err = aws.NewCredentialsCache(cfg.Credentials).Retrieve(ctx)
			if err != nil {
				return aws.Credentials{}, err
			}
		}
	}

	if err := sessionCredStorage.StoreCredentials(c.AWSConfig.Profile, credentials); err != nil {
		clio.Warnf("Error caching credentials, MFA token will be requested before current token is expired")
	}

	// inform the user about using the secure storage to securely store IAM user credentials
	// if it has no parents and it reached this point, it must have had plain text credentials
	// if it has parents, and the root is not a secure storage iam profile, then it has plain text credentials
	if len(c.Parents) == 0 || !c.Parents[0].HasSecureStorageIAMCredentials {
		clio.Warnf("Profile %s has plaintext credentials stored in the AWS credentials file", c.Name)
		clio.Infof("To move the credentials to secure storage, run 'granted credentials import %s'", c.Name)
	}

	return credentials, nil

}

// if required will get a FederationToken to be used to launch the console
// This is required if the iam profile does not assume a role using sts.AssumeRole
func (aia *AwsIamAssumer) AssumeConsole(ctx context.Context, c *Profile, configOpts ConfigOpts) (aws.Credentials, error) {
	if c.AWSConfig.Credentials.SessionToken != "" {
		clio.Debug("found existing session token in credentials for IAM profile, using this to launch the console")
		return c.AWSConfig.Credentials, nil
	} else if c.AWSConfig.RoleARN == "" {
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
func (aia *AwsIamAssumer) ProfileMatchesType(rawProfile *ini.Section, parsedProfile config.SharedConfig) bool {
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

// GetFederationToken is used when launching a console session with long-lived IAM credentials profiles
// GetFederation token uses an allow all IAM policy so that the console session will be able to access everything
// If this is not provided, the session cannot do anything in the console
func getFederationToken(ctx context.Context, c *Profile) (aws.Credentials, error) {
	opts := []func(*config.LoadOptions) error{
		// load the config profile
		config.WithSharedConfigProfile(c.Name),
	}

	// load the creds from the credentials file
	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return aws.Credentials{}, err
	}

	client := sts.NewFromConfig(cfg)

	caller, err := client.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return aws.Credentials{}, err
	}
	name := *caller.UserId

	// for an iam credential, the userID might be something like abcd:test@example.com or in might just be an id
	// the idea here is to use the name portion as the federation token id
	parts := strings.SplitN(*caller.UserId, ":", 2)
	if len(parts) > 1 {
		name = parts[1]
	}
	// name is truncated to ensure it meets the maximum length requirements for the AWS api
	out, err := client.GetFederationToken(ctx, &sts.GetFederationTokenInput{Name: aws.String(truncateString(name, 32)), Policy: aws.String(allowAllPolicy),
		// tags are added to the federation token
		Tags: []types.Tag{
			{Key: aws.String("userID"), Value: caller.UserId},
			{Key: aws.String("account"), Value: caller.Account},
			{Key: aws.String("principalArn"), Value: caller.Arn},
		}})
	if err != nil {
		return aws.Credentials{}, err
	}
	return TypeCredsToAwsCreds(*out.Credentials), err

}

// getTemporaryCreds will call STS to obtain temporary credentials. Will prompt for MFA code.
func (aia *AwsIamAssumer) getTemporaryCreds(ctx context.Context, cfg aws.Config, c *Profile, configOpts ConfigOpts) (aws.Credentials, error) {
	stsClient := sts.New(sts.Options{
		Credentials: cfg.Credentials,
		Region:      c.AWSConfig.Region,
	})

	mfaCode := configOpts.MFATokenCode
	if mfaCode == "" {
		code, err := MfaTokenProvider()
		if err != nil {
			return aws.Credentials{}, err
		}

		mfaCode = code
	}

	sessionTokenOutput, err := stsClient.GetSessionToken(ctx, &sts.GetSessionTokenInput{
		SerialNumber: aws.String(c.AWSConfig.MFASerial),
		TokenCode:    aws.String(mfaCode),
	})

	if err != nil {
		return aws.Credentials{}, err
	}

	newCredentials := aws.Credentials{
		AccessKeyID:     aws.ToString(sessionTokenOutput.Credentials.AccessKeyId),
		SecretAccessKey: aws.ToString(sessionTokenOutput.Credentials.SecretAccessKey),
		SessionToken:    aws.ToString(sessionTokenOutput.Credentials.SessionToken),
		CanExpire:       true,
		Expires:         aws.ToTime(sessionTokenOutput.Credentials.Expiration),
		Source:          aia.Type(),
	}

	return newCredentials, nil
}

func truncateString(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length]
}

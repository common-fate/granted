package cfaws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/common-fate/clio"
	"github.com/common-fate/granted/pkg/securestorage"
	"gopkg.in/ini.v1"
)

type AwsIamMfaAssumer struct {
	AwsIamAssumer
}

func (aia *AwsIamMfaAssumer) AssumeTerminal(ctx context.Context, c *Profile, configOpts ConfigOpts) (aws.Credentials, error) {
	credentials, err := aia.AwsIamAssumer.AssumeTerminal(ctx, c, configOpts)

	// This case is already handled by AwsIamAssumer
	if !c.HasSecureStorageIAMCredentials {
		return credentials, err
	}

	if c.AWSConfig.MFASerial == "" {
		return credentials, nil
	}

	secureSessionCredentialStorage := securestorage.NewSecureSessionCredentialStorage()

	cachedCreds, ok, err := secureSessionCredentialStorage.GetCredentials(c.AWSConfig.Profile)
	if err != nil {
		return cachedCreds, err
	}

	if ok && !cachedCreds.Expired() {
		return cachedCreds, nil
	}

	stsClient := sts.New(sts.Options{
		Credentials: aws.NewCredentialsCache(&CredProv{credentials}),
		Region:      c.AWSConfig.Region,
	})

	mfaCode, err := MfaTokenProvider()
	if err != nil {
		return credentials, err
	}

	sessionTokenOutput, err := stsClient.GetSessionToken(ctx, &sts.GetSessionTokenInput{
		SerialNumber: aws.String(c.AWSConfig.MFASerial),
		TokenCode:    aws.String(mfaCode),
	})

	if err != nil {
		return credentials, err
	}

	newCredentials := aws.Credentials{
		AccessKeyID:     *sessionTokenOutput.Credentials.AccessKeyId,
		SecretAccessKey: *sessionTokenOutput.Credentials.SecretAccessKey,
		SessionToken:    *sessionTokenOutput.Credentials.SessionToken,
		CanExpire:       true,
		Expires:         *sessionTokenOutput.Credentials.Expiration,
		Source:          aia.Type(),
	}

	if err := secureSessionCredentialStorage.StoreCredentials(c.AWSConfig.Profile, newCredentials); err != nil {
		clio.Warnf("Error caching credentials, MFA token will be requested before current token is expired")
	}

	return newCredentials, nil

}

func (aia *AwsIamMfaAssumer) AssumeConsole(ctx context.Context, c *Profile, configOpts ConfigOpts) (aws.Credentials, error) {
	return aia.AwsIamAssumer.AssumeConsole(ctx, c, configOpts)
}

func (aia *AwsIamMfaAssumer) Type() string {
	return "AWS_IAM_MFA"
}

// Matches the profile type on whether it is not an sso profile.
// this will also match other types that are not sso profiles so it should be the last option checked when determining the profile type
func (aia *AwsIamMfaAssumer) ProfileMatchesType(rawProfile *ini.Section, parsedProfile config.SharedConfig) bool {
	return parsedProfile.SSOAccountID == "" && parsedProfile.MFASerial != ""
}

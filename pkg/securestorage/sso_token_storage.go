package securestorage

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
	"github.com/common-fate/clio"
)

type SSOTokensSecureStorage struct {
	SecureStorage SecureStorage
}

func NewSecureSSOTokenStorage() SSOTokensSecureStorage {
	return SSOTokensSecureStorage{
		SecureStorage: SecureStorage{
			StorageSuffix: "aws-sso-tokens",
		},
	}
}

type SSOToken struct {
	// AccessToken is serialized as "AccessToken" to preserve backwards compatibility
	// with earlier versions of Granted. The native AWS CLI serializes this field in camelCase
	// as 'accessToken'. This field key may be changed in future to 'accessToken'.
	AccessToken string `json:"AccessToken"`
	// Expiry is serialized as "Expiry" to preserve backwards compatibility
	// with earlier versions of Granted. The native AWS CLI serializes this field in camelCase
	// as 'expiry'. This field key may be changed in future to 'expiry'.
	Expiry                time.Time `json:"Expiry"`
	ClientID              string    `json:"clientId,omitempty"`
	ClientSecret          string    `json:"clientSecret,omitempty"`
	RegistrationExpiresAt time.Time `json:"registrationExpiresAt,omitempty"`
	Region                string    `json:"region,omitempty"`
	RefreshToken          *string   `json:"refreshToken,omitempty"`
}

// GetValidSSOToken loads and potentially refreshes an AWS SSO access token from secure storage.
// It returns nil if no token was found, or if it is expired
func (s *SSOTokensSecureStorage) GetValidSSOToken(ctx context.Context, profileKey string) *SSOToken {
	var t SSOToken
	err := s.SecureStorage.Retrieve(profileKey, &t)
	if err != nil {
		clio.Debugf("error retrieving IAM Identity Center token from secure storage: %s", err.Error())
		return nil
	}
	now := time.Now()
	isExpired := t.Expiry.Before(now)

	if !isExpired {
		// token is valid
		return &t
	}

	if t.RefreshToken == nil {
		// can't refresh the token, so return nil
		return nil
	}

	if *t.RefreshToken == "" {
		// can't refresh the token, so return nil
		return nil
	}

	// if we get here, we can attempt to refresh the token
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		clio.Errorf("error loading default AWS config for token refresh: %s", err.Error())
		// token is invalid
		return nil
	}

	if t.Region == "" {
		// if the region is not set, the AWS SSO OIDC client will make an invalid API call and will return an
		// 'InvalidGrantException' error.
		clio.Errorf("existing token had no SSO region set")
		// token is invalid
		return nil
	}

	cfg.Region = t.Region

	client := ssooidc.NewFromConfig(cfg)

	res, err := client.CreateToken(ctx, &ssooidc.CreateTokenInput{
		ClientId:     &t.ClientID,
		ClientSecret: &t.ClientSecret,
		GrantType:    aws.String("refresh_token"),
		RefreshToken: t.RefreshToken,
	})
	if err != nil {
		clio.Errorf("error refreshing AWS IAM Identity Center token: %s", err.Error())
		// token is invalid
		return nil
	}

	newToken := SSOToken{
		AccessToken:           *res.AccessToken,
		Expiry:                time.Now().Add(time.Duration(res.ExpiresIn) * time.Second),
		ClientID:              t.ClientID,              // same as the previous token, because the same client was used to refresh
		ClientSecret:          t.ClientSecret,          // same as the previous token, because the same client was used to refresh
		RegistrationExpiresAt: t.RegistrationExpiresAt, // same as the previous token, because the same client was used to refresh
		RefreshToken:          res.RefreshToken,
		Region:                t.Region,
	}

	// save the refreshed token to secure storage
	s.StoreSSOToken(profileKey, newToken)

	return &newToken
}

// Attempts to store the token, any errors will be logged to debug logging
func (s *SSOTokensSecureStorage) StoreSSOToken(profileKey string, ssoTokenValue SSOToken) {
	err := s.SecureStorage.Store(profileKey, ssoTokenValue)
	if err != nil {
		clio.Debugf("writing sso token to credentials cache: %s", err.Error())
	}
}

// Attempts to clear the token, any errors will be logged to debug logging
func (s *SSOTokensSecureStorage) ClearSSOToken(profileKey string) {
	err := s.SecureStorage.Clear(profileKey)
	if err != nil {
		clio.Debugf("clearing sso token from the credentials cache: %s", err)
	}
}

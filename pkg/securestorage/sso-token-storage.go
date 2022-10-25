package securestorage

import (
	"time"

	"github.com/common-fate/clio"
	"github.com/pkg/errors"
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
	AccessToken string
	Expiry      time.Time
}

// GetValidSSOToken returns nil if no token was found, or if it is expired
func (s *SSOTokensSecureStorage) GetValidSSOToken(profileKey string) *SSOToken {
	var t SSOToken
	err := s.SecureStorage.Retrieve(profileKey, &t)
	if err != nil {
		clio.Debug("%s\n", errors.Wrap(err, "GetValidCachedToken").Error())
	}
	if t.Expiry.Before(time.Now()) {
		return nil
	}
	return &t
}

// Attempts to store the token, any errors will be logged to debug logging
func (s *SSOTokensSecureStorage) StoreSSOToken(profileKey string, ssoTokenValue SSOToken) {
	err := s.SecureStorage.Store(profileKey, ssoTokenValue)
	if err != nil {
		clio.Debug("%s\n", errors.Wrap(err, "writing sso token to credentials cache").Error())
	}

}

// Attempts to clear the token, any errors will be logged to debug logging
func (s *SSOTokensSecureStorage) ClearSSOToken(profileKey string) {
	err := s.SecureStorage.Clear(profileKey)
	if err != nil {
		clio.Debug("%s\n", errors.Wrap(err, "clearing sso token from the credentials cache").Error())
	}
}

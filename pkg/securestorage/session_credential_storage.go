package securestorage

import (
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
)

type SessionCredentialSecureStorage struct {
	SecureStorage SecureStorage
}

func NewSecureSessionCredentialStorage() SessionCredentialSecureStorage {
	return SessionCredentialSecureStorage{
		SecureStorage: SecureStorage{
			StorageSuffix: "aws-session-credentials",
		},
	}
}

func (i *SessionCredentialSecureStorage) GetCredentials(profile string) (*aws.Credentials, error) {
	// by default, set the credentials to be expiring so that we force a refresh if there are
	// any problems unmarshalling them.
	credentials := aws.Credentials{
		CanExpire: true,
	}

	err := i.SecureStorage.Retrieve(profile, &credentials)
	if err != nil {
		return nil, err
	}

	return &credentials, nil
}

func (i *SessionCredentialSecureStorage) StoreCredentials(profile string, credentials aws.Credentials) (err error) {
	if credentials.AccessKeyID == "" {
		return errors.New("could not cache credentials: access key ID was empty")
	}
	err = i.SecureStorage.Store(profile, &credentials)
	return
}

package securestorage

import (
	"github.com/99designs/keyring"
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

func (i *SessionCredentialSecureStorage) GetCredentials(profile string) (credentials aws.Credentials, ok bool, err error) {
	err = i.SecureStorage.Retrieve(profile, &credentials)
	if err == keyring.ErrKeyNotFound {
		err = nil
	} else if err == nil {
		ok = true
	}
	return
}

func (i *SessionCredentialSecureStorage) StoreCredentials(profile string, credentials aws.Credentials) (err error) {
	err = i.SecureStorage.Store(profile, &credentials)
	return
}

package securestorage

import "github.com/aws/aws-sdk-go-v2/aws"

type IAMCredentialsSecureStorage struct {
	SecureStorage SecureStorage
}

func NewSecureIAMCredentialStorage() IAMCredentialsSecureStorage {
	return IAMCredentialsSecureStorage{
		SecureStorage: SecureStorage{
			StorageSuffix: "aws-iam-credentials",
		},
	}
}

func (i *IAMCredentialsSecureStorage) GetCredentials(profile string) (credentials aws.Credentials, err error) {
	err = i.SecureStorage.Retrieve(profile, &credentials)
	return
}

func (i *IAMCredentialsSecureStorage) StoreCredentials(profile string, credentials aws.Credentials) (err error) {
	err = i.SecureStorage.Store(profile, &credentials)
	return
}

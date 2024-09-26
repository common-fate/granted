package securestorage

import (
	"time"

	"github.com/99designs/keyring"
	"github.com/common-fate/clio"
)

type DeviceCodeSecureStorage struct {
	SecureStorage SecureStorage
}

type AllUserCodes struct {
	Codes []UserCode `json:"codes"`
}

// Prune stale user codes that have expired.
func (a *AllUserCodes) Prune(now time.Time) {
	var valid []UserCode
	for _, c := range a.Codes {
		if c.Expiry.After(now) {
			valid = append(valid, c)
		}
	}

	a.Codes = valid
}

type UserCode struct {
	Code   string    `json:"code"`
	Expiry time.Time `json:"expiry"`
}

func NewDeviceCodeSecureStorage() DeviceCodeSecureStorage {
	return DeviceCodeSecureStorage{
		SecureStorage: SecureStorage{
			StorageSuffix: "aws-idc-user-codes",
		},
	}
}

func (i *DeviceCodeSecureStorage) GetValidUserCodes() (AllUserCodes, error) {
	var u AllUserCodes
	err := i.SecureStorage.Retrieve("codes", &u)
	if err == keyring.ErrKeyNotFound {
		return AllUserCodes{}, nil
	}

	if err != nil {
		return AllUserCodes{}, err
	}

	u.Prune(time.Now())

	clio.Debugw("retrieve user codes from keychain", "codes", u)

	return u, nil
}

func (i *DeviceCodeSecureStorage) StoreUserCode(code UserCode) (err error) {
	allCodes, err := i.GetValidUserCodes()
	if err != nil {
		clio.Debugf("storeusercode: error retrieving IDC user codes in keychain: %s", err.Error())
	}

	allCodes.Codes = append(allCodes.Codes, code)

	clio.Debugw("storing user codes in keychain", "codes", allCodes)

	err = i.SecureStorage.Store("codes", &allCodes)
	return
}

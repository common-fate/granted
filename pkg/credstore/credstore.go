package credstore

import (
	"encoding/json"
	"errors"
	"os"
	"path"

	"github.com/99designs/keyring"
	"github.com/AlecAivazis/survey/v2"
	"github.com/common-fate/granted/pkg/config"
	"github.com/common-fate/granted/pkg/testable"
)

var ErrCouldNotOpenKeyring error = errors.New("keyring not opened successfully")

// returns ring.ErrKeyNotFound if not found
func Retrieve(key string, target interface{}) error {
	ring, err := openKeyring()
	if err != nil {
		return err
	}
	keyringItem, err := ring.Get(key)
	if err != nil {
		return err
	}
	return json.Unmarshal(keyringItem.Data, &target)
}

func Store(key string, payload interface{}) error {
	ring, err := openKeyring()
	if err != nil {
		return err
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return ring.Set(keyring.Item{
		Key:  key, // store with the corresponding key
		Data: b,   // store the bytes
	})
}

func Clear(key string) error {
	ring, err := openKeyring()
	if err != nil {
		return err
	}
	return ring.Remove(key)
}

func openKeyring() (keyring.Keyring, error) {
	grantedFolder, err := config.GrantedConfigFolder()
	if err != nil {
		return nil, err
	}
	// check if the cred-store file exists in the folder
	credStorePath := path.Join(grantedFolder, "cred-store")

	ring, err := keyring.Open(keyring.Config{
		AllowedBackends: []keyring.BackendType{keyring.FileBackend},
		FileDir:         credStorePath,
		FilePasswordFunc: func(s string) (string, error) {
			in := survey.Password{Message: s}
			var out string
			withStdio := survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)
			err := testable.AskOne(&in, &out, withStdio)
			return out, err
		},
		ServiceName: "granted",
	})
	if err != nil {
		return nil, err
	}
	// Ensure this keyring is correctly opened, there have been some issues with ring being nil with no error
	if ring == nil {
		return nil, ErrCouldNotOpenKeyring
	}
	return ring, err
}

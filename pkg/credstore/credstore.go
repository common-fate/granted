package credstore

import (
	"encoding/json"
	"os"
	"path"

	"github.com/99designs/keyring"
	"github.com/common-fate/granted/pkg/config"
	"github.com/common-fate/granted/pkg/debug"
)

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
		return nil
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
		return nil
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
	_, err = os.Stat(credStorePath)
	fileExists := err == nil

	if !fileExists {
		debug.Fprintf(debug.VerbosityDebug, os.Stderr, "ℹ️  A cred-store file was not found\n")
		debug.Fprintf(debug.VerbosityDebug, os.Stderr, "Creating cred-store file at %s\n", credStorePath)
		_, err = os.Create(credStorePath)
		if err != nil {
			return nil, err
		}

	}
	return keyring.Open(keyring.Config{
		FileDir:     path.Join(grantedFolder, "cred-store"),
		ServiceName: "granted",
	})

}

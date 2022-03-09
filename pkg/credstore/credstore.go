package credstore

import (
	"encoding/json"
	"errors"
	"os"
	"path"
	"strings"

	"github.com/99designs/keyring"
	"github.com/AlecAivazis/survey/v2"
	"github.com/common-fate/granted/pkg/config"
	"github.com/common-fate/granted/pkg/debug"
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

func Store(key string, payload interface{}, profile string) error {
	ring, err := openKeyring()
	if err != nil {
		return err
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	//look up the keyring to see if we already have a corresponding key
	//if theres a url already set then update the description for another profile
	var desc string
	keyringItem, err := ring.Get(key)
	if err != nil {
		desc = profile
		debug.Fprintf(debug.VerbosityDebug, os.Stderr, "key not found")
	} else {
		if !strings.Contains(keyringItem.Description, profile) {
			desc = keyringItem.Description + ", " + profile

		}
	}

	return ring.Set(keyring.Item{
		Key:         key,  // store with the corresponding key
		Data:        b,    // store the bytes
		Description: desc, //save the name for readability
	})
}

func Clear(key string) error {
	ring, err := openKeyring()
	if err != nil {
		return err
	}
	return ring.Remove(key)
}

func ClearAll() error {

	ring, err := openKeyring()
	if err != nil {
		return err
	}
	keys, err := ring.Keys()
	if err != nil {
		return err
	}
	for _, k := range keys {
		err := ring.Remove(k)
		if err != nil {
			return err
		}

	}
	return nil
}

func openKeyring() (keyring.Keyring, error) {
	grantedFolder, err := config.GrantedConfigFolder()
	if err != nil {
		return nil, err
	}
	// check if the cred-store file exists in the folder
	credStorePath := path.Join(grantedFolder, "cred-store")

	return keyring.Open(keyring.Config{
		FileDir: credStorePath,
		FilePasswordFunc: func(s string) (string, error) {
			in := survey.Password{Message: s}
			var out string
			withStdio := survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)
			err := testable.AskOne(&in, &out, withStdio)
			return out, err
		},
		ServiceName: "granted",
	})
}

func List() ([]keyring.Item, error) {
	tokenList := []keyring.Item{}
	ring, err := openKeyring()
	if err != nil {
		return nil, err
	}
	keys, err := ring.Keys()
	if err != nil {
		return nil, err
	}
	for _, k := range keys {
		item, err := ring.Get(k)
		if err != nil {
			return nil, err
		}
		tokenList = append(tokenList, item)

	}
	return tokenList, nil
}

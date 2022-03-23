package credstore

import (
	"encoding/json"
	"os"
	"path"

	"github.com/99designs/keyring"
	"github.com/AlecAivazis/survey/v2"
	"github.com/common-fate/granted/pkg/config"
	"github.com/common-fate/granted/pkg/debug"
	"github.com/common-fate/granted/pkg/testable"
	"github.com/pkg/errors"
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
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	grantedFolder, err := config.GrantedConfigFolder()
	if err != nil {
		return nil, err
	}

	credStorePath := path.Join(grantedFolder, "cred-store")

	c := keyring.Config{
		ServiceName: "granted",

		// MacOS keychain
		KeychainName:             "login",
		KeychainTrustApplication: true,

		// KDE Wallet
		KWalletAppID:  "granted",
		KWalletFolder: "granted",

		// Windows
		WinCredPrefix: "granted",

		// freedesktop.org's Secret Service
		LibSecretCollectionName: "granted",

		// Pass (https://www.passwordstore.org/)
		PassPrefix: "granted",

		// Fallback encrypted file
		FileDir: credStorePath,
		FilePasswordFunc: func(s string) (string, error) {
			in := survey.Password{Message: s}
			var out string
			withStdio := survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)
			err := testable.AskOne(&in, &out, withStdio)
			return out, err
		},
	}

	// enable debug logging if the verbose flag is set in the CLI
	if debug.CliVerbosity == debug.VerbosityDebug {
		keyring.Debug = true
	}

	if cfg.Keyring != nil {
		if cfg.Keyring.Backend != nil {
			c.AllowedBackends = []keyring.BackendType{keyring.BackendType(*cfg.Keyring.Backend)}
		}
		if cfg.Keyring.KeychainName != nil {
			c.KeychainName = *cfg.Keyring.KeychainName
		}
		if cfg.Keyring.FileDir != nil {
			c.FileDir = *cfg.Keyring.FileDir
		}
		if cfg.Keyring.LibSecretCollectionName != nil {
			c.LibSecretCollectionName = *cfg.Keyring.LibSecretCollectionName
		}
	}

	k, err := keyring.Open(c)
	if err != nil {
		return nil, errors.Wrap(err, "opening keyring")
	}

	return k, nil
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

func ListKeys() ([]string, error) {
	ring, err := openKeyring()
	if err != nil {
		return nil, err
	}
	return ring.Keys()
}

package cfaws

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/99designs/keyring"
	"github.com/common-fate/granted/pkg/config"
)

// This file defines two methods for interacting with the credential store
//
// 1. CheckCredStore
// 2. WriteSSOCreds
//
// The first method is used to check if the credential store exists and if it does,
// it will attempt to get the credentials from the keyring.
//
// The second method is used to write the credentials to the keyring.
//

// NOTE: we need to validate if using profile name as a key has any other implications (clashes)

type SSOToken struct {
	AccessToken string
	Expiry      time.Time
}

var ErrNoCachedCredentials error = errors.New("no cached credentials")

// CheckSSOTokenStore returns ErrNoCachedCredentials if the credentials are not cached or they are expired
func CheckSSOTokenStore(profileKey string) (*SSOToken, error) {
	ring, err := openKeyring()
	if err != nil {
		return nil, err
	}

	keyringItem, err := ring.Get(profileKey)
	if err != nil {
		return nil, ErrNoCachedCredentials
	}
	var token *SSOToken
	err = json.Unmarshal(keyringItem.Data, &token)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding: %s\n", err)
	}

	if token.Expiry.Before(time.Now()) {
		return nil, ErrNoCachedCredentials
	}
	return token, nil
}

// Testing fn for cred stores
func WriteSSOToken(profileKey string, ssoTokenValue SSOToken) error {
	ring, err := openKeyring()
	if err != nil {
		return nil
	}

	// encode ssoTokenValue struct as raw bytes []byte
	encodedCredentials, err := json.Marshal(ssoTokenValue)
	if err != nil {
		return err
	}

	err = ring.Set(keyring.Item{
		Key:  profileKey,         // store with the corresponding profileKey
		Data: encodedCredentials, // store the byte encoded creds
	})
	if err != nil {
		return err
	}

	return nil
}

// Clear a token if required
func ClearSSOToken(profileKey string) error {
	ring, err := openKeyring()
	if err != nil {
		return nil
	}
	return ring.Remove(profileKey)
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
		if os.Getenv("DEBUG") == "true" {
			fmt.Fprintln(os.Stderr, "ℹ️  A cred-store file was not found")
			fmt.Fprintf(os.Stderr, "Creating cred-store file at %s\n", credStorePath)
		}
		_, err = os.Create(credStorePath)
		if err != nil {
			return nil, err
		}

	}

	// @TOCHECK: I'm unsure how the keyring package is encrypting the data (if at all). May need further configuration
	return keyring.Open(keyring.Config{
		FileDir:     path.Join(grantedFolder, "cred-store"),
		ServiceName: "granted",
	})

}

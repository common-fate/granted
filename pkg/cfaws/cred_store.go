package cfaws

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
	"path"

	"github.com/99designs/keyring"
	"github.com/aws/aws-sdk-go-v2/aws"
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
func CheckCredStore(profileKey string) (creds aws.Credentials, err error) {

	grantedFolder, err := config.GrantedConfigFolder()
	if err != nil {
		return aws.Credentials{}, err
	}

	// check if the cred-store file exists in the folder
	credStorePath := path.Join(grantedFolder, "cred-store")
	_, err = os.Stat(credStorePath)
	fileExists := err == nil

	if fileExists {
		// if the file exists, we'll try to get the creds from the keyring
		ring, _ := keyring.Open(keyring.Config{
			FileDir:     path.Join(grantedFolder, "cred-store"),
			ServiceName: "example",
		})

		keyringItem, err := ring.Get(profileKey)
		if err != nil {
			// log specific warning here
			return aws.Credentials{}, err
		}

		rawBytes := keyringItem.Data

		decodedCreds, err := StructBytesDecode(rawBytes)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding: %s\n", err)
		}

		// fmt.Printf(keyringItem.Key)
		// cast the raw bytes to our aws.Credentials type
		// encode aws.Credentials{} struct as raw bytes []byte
		// encoded :=

		return decodedCreds, nil

	} else {
		// default return
		return aws.Credentials{}, err
	}
}

// Testing fn for cred stores
func WriteSSOCreds(key string, ssoTokenValue aws.Credentials) error {

	// encode ssoTokenValue struct as raw bytes []byte
	// encoded :=

	grantedFolder, err := config.GrantedConfigFolder()
	if err != nil {
		return err
	}

	// check if the cred-store file exists in the folder
	credStorePath := path.Join(grantedFolder, "cred-store")
	_, err = os.Stat(credStorePath)
	fileExists := err == nil

	if !fileExists {
		fmt.Fprintln(os.Stdout, "ℹ️  A cred-store file was not found")
		fmt.Fprintf(os.Stdout, "Creating cred-store file at %s\n", credStorePath)
		os.Create(credStorePath)
	}

	// @TOCHECK: I'm unsure how the keyring package is encrypting the data (if at all). May need further configuration
	ring, _ := keyring.Open(keyring.Config{
		FileDir:     path.Join(grantedFolder, "cred-store"),
		ServiceName: "example",
	})

	_ = ring.Set(keyring.Item{
		Key:  "foo",
		Data: []byte("secret-bar"),
	})

	i, _ := ring.Get("foo")

	fmt.Printf("%s", i.Data)

	return nil
}

func StructBytesEncode(ssoTokenValue aws.Credentials) ([]byte, error) {
	// Initialize the encoder and decoder.  Normally enc and dec would be
	// bound to network connections and the encoder and decoder would
	// run in different processes.
	var network bytes.Buffer        // Stand-in for a network connection
	enc := gob.NewEncoder(&network) // Will write to network.

	// encode ssoTokenValue struct as raw bytes []byte
	err := enc.Encode(ssoTokenValue)

	if err != nil {
		// @TODO: Verify what error format to use
		fmt.Fprintf(os.Stderr, "Error encoding: %s\n", err)
	}

	return network.Bytes(), nil
}

func StructBytesDecode(encoded []byte) (aws.Credentials, error) {
	var network bytes.Buffer        // Stand-in for a network connection
	network.Write(encoded)          // Add the encoded bytes to the network buffer
	dec := gob.NewDecoder(&network) // Will read from network.

	// Decode the data in the encoded bytes
	var creds aws.Credentials
	err := dec.Decode(&creds)

	if err != nil {
		return aws.Credentials{}, err
	}

	return creds, nil
}

// package config stores configuration around
// user onboarding to granted used to display friendly
// CLI hints and save progress in multi-step workflows,
// such as deploying Granted services to a user's cloud
// environment.
package config

import (
	"os"
	"path"
	"runtime"
	"time"

	"github.com/BurntSushi/toml"

	"github.com/common-fate/granted/internal/build"
)

type Config struct {
	DefaultBrowser string
	// used to override the builtin filepaths for custom installation locations
	CustomBrowserPath      string
	CustomSSOBrowserPath   string
	LastCheckForUpdates    time.Weekday
	Keyring                *KeyringConfig `toml:",omitempty"`
	Ordering               string
	ExportCredentialSuffix string
	// This is used to generate reference URls
	GrantedApprovalsUrl string
}

type KeyringConfig struct {
	Backend                 *string `toml:",omitempty"`
	KeychainName            *string `toml:",omitempty"`
	FileDir                 *string `toml:",omitempty"`
	LibSecretCollectionName *string `toml:",omitempty"`
}

// NewDefaultConfig returns a config with OS specific defaults populated
func NewDefaultConfig() Config {
	// macos devices should default to the keychain backend
	if runtime.GOOS == "darwin" {
		keychain := "keychain"
		return Config{
			Keyring: &KeyringConfig{
				Backend: &keychain,
			},
		}
	}
	return Config{}
}

// checks and or creates the config folder on startup
func SetupConfigFolder() error {
	grantedFolder, err := GrantedConfigFolder()
	if err != nil {
		return err
	}
	if _, err := os.Stat(grantedFolder); os.IsNotExist(err) {
		err := os.Mkdir(grantedFolder, 0700)
		if err != nil {
			return err
		}
	}
	return nil
}

// checks and or creates the config folder on startup
func SetupZSHAutoCompleteFolderAssume() (string, error) {
	grantedFolder, err := GrantedConfigFolder()
	if err != nil {
		return "", err
	}
	zshPath := path.Join(grantedFolder, "zsh_autocomplete")
	if _, err := os.Stat(zshPath); os.IsNotExist(err) {
		err := os.Mkdir(zshPath, 0700)
		if err != nil {
			return "", err
		}
	}
	zshPath = path.Join(zshPath, build.AssumeScriptName())
	if _, err := os.Stat(zshPath); os.IsNotExist(err) {
		err := os.Mkdir(zshPath, 0700)
		if err != nil {
			return "", err
		}
	}
	return zshPath, nil
}

// checks and or creates the config folder on startup
func SetupZSHAutoCompleteFolderGranted() (string, error) {
	grantedFolder, err := GrantedConfigFolder()
	if err != nil {
		return "", err
	}
	zshPath := path.Join(grantedFolder, "zsh_autocomplete")
	if _, err := os.Stat(zshPath); os.IsNotExist(err) {
		err := os.Mkdir(zshPath, 0700)
		if err != nil {
			return "", err
		}
	}
	zshPath = path.Join(zshPath, build.GrantedBinaryName())
	if _, err := os.Stat(zshPath); os.IsNotExist(err) {
		err := os.Mkdir(zshPath, 0700)
		if err != nil {
			return "", err
		}
	}
	return zshPath, nil
}

func GrantedConfigFolder() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	// check if the .granted folder already exists
	return path.Join(home, build.ConfigFolderName), nil
}

func Load() (*Config, error) {
	grantedFolder, err := GrantedConfigFolder()
	if err != nil {
		return nil, err
	}
	configFilePath := path.Join(grantedFolder, "config")

	file, err := os.OpenFile(configFilePath, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	c := NewDefaultConfig()

	_, err = toml.NewDecoder(file).Decode(&c)
	if err != nil {
		// if there is an error just reset the file
		return &c, nil
	}
	return &c, nil
}

func (store *Config) Save() error {

	grantedFolder, err := GrantedConfigFolder()
	if err != nil {
		return err
	}
	configFilePath := path.Join(grantedFolder, "config")

	file, err := os.OpenFile(configFilePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer file.Close()
	return toml.NewEncoder(file).Encode(store)
}

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

	"github.com/BurntSushi/toml"

	"github.com/common-fate/granted/internal/build"
)

type Config struct {
	DefaultBrowser string
	// used to override the builtin filepaths for custom installation locations
	CustomBrowserPath      string
	CustomSSOBrowserPath   string
	Keyring                *KeyringConfig `toml:",omitempty"`
	Ordering               string
	ExportCredentialSuffix string
	// AccessRequestURL is a Granted Approvals URL that users can visit
	// to request access, in the event that we receive a ForbiddenException
	// denying access to assume a particular role.
	//Set this to true to set `--export` to ~/.aws/credentials as default
	ExportCredsToAWS bool `toml:",omitempty"`

	AccessRequestURL string `toml:",omitempty"`

	CommonFateDefaultSSOStartURL string `toml:",omitempty"`
	CommonFateDefaultSSORegion   string `toml:",omitempty"`

	// Set this to true to disable usage tips like `To assume this profile again later without needing to select it, run this command:`
	DisableUsageTips bool `toml:",omitempty"`
	// Set this to true to disable credential caching feature when using credential process
	DisableCredentialProcessCache bool `toml:",omitempty"`

	// Set this to true to set `--export-all-env-vars` as default
	DefaultExportAllEnvVar bool `toml:",omitempty"`

	// deprecated in favor of ProfileRegistry
	ProfileRegistryURLS []string `toml:",omitempty"`
	ProfileRegistry     struct {
		// add any global configuration to profile registry here.
		PrefixAllProfiles       bool
		PrefixDuplicateProfiles bool
		SessionName             string            `toml:",omitempty"`
		RequiredKeys            map[string]string `toml:",omitempty"`
		Variables               map[string]string `toml:",omitempty"`
		Registries              []Registry        `toml:",omitempty"`
	} `toml:",omitempty"`
}

type KeyringConfig struct {
	Backend                 *string `toml:",omitempty"`
	KeychainName            *string `toml:",omitempty"`
	FileDir                 *string `toml:",omitempty"`
	LibSecretCollectionName *string `toml:",omitempty"`
}

type Registry struct {
	Name                    string  `toml:"name"`
	URL                     string  `toml:"url"`
	Path                    *string `toml:"path,omitempty"`
	Filename                *string `toml:"filename,omitempty"`
	Ref                     *string `toml:"ref,omitempty"`
	Priority                *int    `toml:"priority, omitempty"`
	PrefixDuplicateProfiles bool    `toml:"prefixDuplicateProfiles,omitempty"`
	PrefixAllProfiles       bool    `toml:"prefixAllProfiles,omitempty"`
	WriteOnSyncFailure      bool    `toml:"writeOnSyncFailure,omitempty"`
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

func (c *Config) Save() error {
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
	return toml.NewEncoder(file).Encode(c)
}

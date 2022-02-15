// package config stores configuration around
// user onboarding to granted used to display friendly
// CLI hints and save progress in multi-step workflows,
// such as deploying Granted services to a user's cloud
// environment.
package config

import (
	"os"
	"path"

	"github.com/common-fate/granted/internal/build"
)

func GrantedConfigFolder() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	// check if the .granted folder already exists
	return path.Join(home, build.ConfigFolderName), nil
}

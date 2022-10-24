package launcher

import (
	"strings"
)

type ChromeProfile struct {
	// ExecutablePath is the path to the Chrome binary on the system.
	ExecutablePath string
	// UserDataPath is the path to the Chrome user data directory,
	// which we override to put Granted profiles in a specific folder
	// for easy management.
	UserDataPath string
}

func (l ChromeProfile) LaunchCommand(url string, profile string) []string {
	profile = strings.ReplaceAll(profile, " ", "_")
	return []string{
		l.ExecutablePath,
		"--user-data-dir=" + l.UserDataPath,
		"--profile-directory=granted-profile-" + profile,
		"--no-first-run",
		"--no-default-browser-check",
		url,
	}
}

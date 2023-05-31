package launcher

import (
	"fmt"
	"hash/fnv"
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
	profileName := chromeProfileName(profile)
	return []string{
		l.ExecutablePath,
		"--user-data-dir=" + l.UserDataPath,
		"--profile-directory=" + profileName,
		"--no-first-run",
		"--no-default-browser-check",
		url,
	}
}

func chromeProfileName(profile string) string {
	h := fnv.New32a()
	h.Write([]byte(profile))

	hash := fmt.Sprint(h.Sum32())
	return hash
}

func (l ChromeProfile) UseForkProcess() bool { return true }

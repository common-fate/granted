package launcher

import (
	"encoding/json"
	"os"
	"path"
	"runtime"

	"github.com/common-fate/clio"
	"github.com/common-fate/granted/pkg/browser"
	"github.com/pkg/errors"
)

type ChromeProfile struct {
	// ExecutablePath is the path to the Chrome binary on the system.
	ExecutablePath string
	// UserDataPath is the path to the Chrome user data directory,
	// which we override to put Granted profiles in a specific folder
	// for easy management.
	UserDataPath string

	BrowserType string
}

func (l ChromeProfile) LaunchCommand(url string, profile string) []string {
	profileName := FindBrowserProfile(profile, l.BrowserType)

	return []string{
		l.ExecutablePath,
		// "--user-data-dir=" + l.UserDataPath,
		"--profile-directory=" + profileName,
		"--no-first-run",
		"--no-default-browser-check",
		url,
	}
}

var BravePathMac = "Library/Application Support/BraveSoftware/Brave-Browser/Local State"
var BravePathLinux = ".config/brave-browser/Local State"
var BravePathWindows = `AppData\Local\BraveSoftware\Brave-Browser\Local State`

var ChromePathMac = "Library/Application Support/Google/Chrome/Local State"
var ChromePathLinux = ".config/google-chrome/Local State"
var ChromePathWindows = `AppData\Local\Google\Chrome\User Data/Local State`

var EdgePathMac = `Library/Application Support/Microsoft\ Edge/Local State`
var EdgePathLinux = ".config/microsoft-edge/Local State"
var EdgePathWindows = `AppData\Local\Microsoft Edge\User Data/Local State`

var ChromiumPathMac = "Library/Application Support/Chromium/Local State"
var ChromiumPathLinux = ".config/chromium/Local State"
var ChromiumPathWindows = `AppData\Local\Chromium\User Data/Local State`

// FindBrowserProfile will try to read profile data from local state path.
// will fallback to provided profile value.
func FindBrowserProfile(profile string, browserType string) string {
	// work out which chromium browser we are using
	stateFile, err := getLocalStatePath(browserType)
	if err != nil {
		clio.Debugf("unable to find localstate path with err %s", err)
		return profile
	}

	//read the state file
	data, err := os.ReadFile(stateFile)
	if err != nil {
		clio.Debugf("unable to read local state file with err %s", err)
		return profile
	}

	//the Local State json blob is a bunch of map[string]interfaces which makes it difficult to unmarshal
	var f map[string]interface{}
	err = json.Unmarshal(data, &f)
	if err != nil {
		clio.Debugf("unable to unmarshal local state file with err %s", err)
		return profile
	}

	//grab the profiles out from the json blob
	profiles := f["profile"].(map[string]interface{})
	//can this be done cleaner with a conversion into a struct?
	for profileName, profileObj := range profiles["info_cache"].(map[string]interface{}) {
		//if the profile name is the same as the profile name we are assuming then we want to use the same profile
		if profileObj.(map[string]interface{})["name"] == profile {
			return profileName
		}

	}

	return profile
}

func getLocalStatePath(browserType string) (stateFile string, err error) {
	stateFile, err = os.UserHomeDir()
	if err != nil {
		return "", err
	}
	switch runtime.GOOS {
	case "windows":
		switch browserType {
		case browser.ChromeKey:
			stateFile = path.Join(stateFile, ChromePathWindows)

		case browser.BraveKey:
			stateFile = path.Join(stateFile, BravePathWindows)

		case browser.EdgeKey:
			stateFile = path.Join(stateFile, EdgePathWindows)

		case browser.ChromiumKey:
			stateFile = path.Join(stateFile, ChromiumPathWindows)
		}

	case "darwin":
		switch browserType {
		case browser.ChromeKey:
			stateFile = path.Join(stateFile, ChromePathMac)

		case browser.BraveKey:
			stateFile = path.Join(stateFile, BravePathMac)

		case browser.EdgeKey:
			stateFile = path.Join(stateFile, EdgePathMac)

		case browser.ChromiumKey:
			stateFile = path.Join(stateFile, ChromiumPathMac)
		}

	case "linux":
		switch browserType {
		case browser.ChromeKey:
			stateFile = path.Join(stateFile, ChromePathLinux)

		case browser.BraveKey:
			stateFile = path.Join(stateFile, BravePathLinux)

		case browser.EdgeKey:
			stateFile = path.Join(stateFile, EdgePathLinux)

		case browser.ChromiumKey:
			stateFile = path.Join(stateFile, ChromiumPathLinux)
		}

	default:
		clio.Debug("getting local state path: os not supported")
		return "", errors.New("os not supported")
	}
	return stateFile, nil
}

func (l ChromeProfile) UseForkProcess() bool { return true }

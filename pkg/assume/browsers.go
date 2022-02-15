package assume

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"runtime"

	"github.com/common-fate/granted/pkg/config"
	"github.com/pkg/browser"
)

const ChromePathMac = "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"

// @TODO confirm this works
const ChromePathLinux = `/usr/bin/google-chrome`

// @TODO confirm this works
const ChromePathWindows = `%ProgramFiles%\Google\Chrome\Application\chrome.exe`

// Only supports mac
func OpenWithChromeProfile(url string, labels RoleLabels) error {
	opSys := runtime.GOOS
	chromePath := ""
	switch opSys {
	case "windows":
		chromePath = ChromePathWindows
	case "darwin":
		chromePath = ChromePathMac
	case "linux":
		chromePath = ChromePathLinux
	default:
		return errors.New("os not supported")
	}

	// check if the default chrome location is accessible
	_, err := os.Stat(chromePath)
	if err == nil {

		grantedFolder, err := config.GrantedConfigFolder()
		if err != nil {
			return err
		}
		// Creates and/or opens a chrome browser with a new profile
		// profiles will be stored in the the %HOME%/.granted directory
		// unfortunately, the profiles will be created with the name as "Person x"
		// The only way to programatically rename the profile is to open chrome with a new profile, close chrome then edit the Preferences.json file profile.name property, then reopen chrome
		// A possible approach would be to open chrome in a headless way first then open it fully after setting the name
		profile := fmt.Sprintf("%s:=%s", labels.Role, labels.Account)
		userDataPath := path.Join(grantedFolder, "chrome-profiles")
		cmd := exec.Command(chromePath,
			fmt.Sprintf("--user-data-dir=%s", userDataPath), "--profile-directory="+profile, "--no-first-run", "--no-default-browser-check", url,
		)
		err = cmd.Start()
		if err != nil {
			return err
		}
		// detach from this new process because it continues to run
		return cmd.Process.Release()
	}
	return errors.New("could not locate a Chrome installation")

}

const FirefoxPathMac = "/Applications/Firefox.app/Contents/MacOS/firefox"

// @TODO confirm this works
const FirefoxPathLinux = `/usr/bin/firefox`

// @TODO confirm this works
const FirefoxPathWindows = `%ProgramFiles%\Mozilla Firefox\firefox.exe`

func OpenWithFirefoxContainer(urlString string, labels RoleLabels) error {
	opSys := runtime.GOOS
	firefoxPath := ""
	switch opSys {
	case "windows":
		firefoxPath = FirefoxPathWindows
	case "darwin":
		firefoxPath = FirefoxPathMac
	case "linux":
		firefoxPath = FirefoxPathLinux
	default:
		return errors.New("os not supported")
	}

	tabURL := fmt.Sprintf("ext+granted-containers:name=%s:%s (ap-southeast-2)&url=%s", labels.Role, labels.Account, url.QueryEscape(urlString))
	cmd := exec.Command(firefoxPath,
		"--new-tab",
		tabURL)
	err := cmd.Start()
	if err != nil {
		return err
	}
	// detach from this new process because it continues to run
	return cmd.Process.Release()

}

type Session struct {
	SessionID    string `json:"sessionId"`
	SesssionKey  string `json:"sessionKey"`
	SessionToken string `json:"sessionToken"`
}
type RoleLabels struct {
	// the name of the role
	Role string
	// a sting which helps to indentify this role to the user
	Account string
}

type Browser int

const (
	BrowerFirefox Browser = iota
	BrowserChrome
	BrowserDefault
)

func LaunchConsoleSession(sess Session, labels RoleLabels, webBrowser Browser) error {
	sessJSON, err := json.Marshal(sess)
	if err != nil {
		return err
	}

	u := url.URL{
		Scheme: "https",
		Host:   "signin.aws.amazon.com",
		Path:   "/federation",
	}
	q := u.Query()
	q.Add("Action", "getSigninToken")
	q.Add("Session", string(sessJSON))
	u.RawQuery = q.Encode()

	res, err := http.Get(u.String())
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("opening console failed with code %v", res.StatusCode)
	}

	token := struct {
		SigninToken string `json:"SigninToken"`
	}{}

	err = json.NewDecoder(res.Body).Decode(&token)
	if err != nil {
		return err
	}

	u = url.URL{
		Scheme: "https",
		Host:   "signin.aws.amazon.com",
		Path:   "/federation",
	}
	q = u.Query()
	q.Add("Action", "login")
	q.Add("Issuer", "")
	q.Add("Destination", "https://console.aws.amazon.com/console/home")
	q.Add("SigninToken", token.SigninToken)
	u.RawQuery = q.Encode()

	switch webBrowser {
	case BrowerFirefox:
		return OpenWithFirefoxContainer(u.String(), labels)
	case BrowserChrome:
		return OpenWithChromeProfile(u.String(), labels)
	default:
		return browser.OpenURL(u.String())
	}
}

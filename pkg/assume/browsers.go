package assume

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/common-fate/granted/pkg/config"
	"github.com/common-fate/granted/pkg/testable"
	"github.com/fatih/color"
	"github.com/pkg/browser"
	"github.com/urfave/cli/v2"
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

type plist struct {
	//XMLName xml.Name `xml:"plist"`
	Pdict Pdict `xml:"dict"`
}

type Pdict struct {
	//XMLName xml.Name `xml:"dict"`
	Key   string `xml:"key"`
	Array Array  `xml:"array"`
}

type Array struct {
	//XMLName xml.Name `xml:"array"`
	Dict Dict `xml:"dict"`
}

type Dict struct {
	//XMLName xml.Name `xml:"dict"`
	Key     []string `xml:"key"`
	Dict    IntDict  `xml:"dict"`
	Strings []string `xml:"string"`
}

type IntDict struct {
	//XMLName xml.Name `xml:"dict"`
	Key     string `xml:"key"`
	Strings string `xml:"string"`
}

//Checks the config to see if the user has already set up their default browser
func UserHasDefaultBrowser(ctx *cli.Context) (bool, error) {
	return false, nil
}

func handleOSXBrowserSearch() (string, error) {
	//get home dir
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	path := home + "/Library/Preferences/com.apple.LaunchServices/com.apple.launchservices.secure.plist"

	//convert plist to xml using putil
	//plutil -convert xml1
	args := []string{"-convert", "xml1", path}
	cmd := exec.Command("plutil", args...)
	err = cmd.Run()
	if err != nil {
		return "", err
	}

	//read plist file
	data, err := ioutil.ReadFile(path)

	if err != nil {
		return "", err
	}
	plist := &plist{}

	// fmt.Fprintf(os.Stderr, "\n%s\n", data)
	//unmarshal the xml into the structs
	err = xml.Unmarshal([]byte(data), &plist)
	if err != nil {
		return "", err
	}

	//get out the default browser

	for i, s := range plist.Pdict.Array.Dict.Strings {
		if s == "http" {
			return plist.Pdict.Array.Dict.Strings[i-1], nil
		}
	}
	return "", nil
}

func handleLinuxBrowserSearch() (string, error) {
	out, err := exec.Command("xdg-settings", "get", "default-web-browser").Output()

	if err != nil {
		return "", err
	}

	return string(out), nil
}

func handleWindowsBrowserSearch() (string, error) {
	return "", nil
}

func handleManualBrowserSelection() (string, error) {
	//didn't find it request manual input

	fmt.Fprintf(os.Stderr, "ℹ️  Could not find default browser\n")
	fmt.Fprintf(os.Stderr, "ℹ️  Select your default browser\n")

	withStdio := survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)
	in := survey.Select{
		Options: []string{"Chromium Browser (Chrome, Edge, Brave)", "Firefox", "Internet Explorer"},
	}
	var roleacc string
	err := testable.AskOne(&in, &roleacc, withStdio)
	if err != nil {
		return "", err
	}

	if roleacc != "" {
		fmt.Fprintf(os.Stderr, "%s\n", roleacc)
		return roleacc, nil
	}
	return "", nil
}

//finds out which browser the use has as default
func FindDefaultBrowser() (string, error) {
	// @TODO confirm this works
	ops := runtime.GOOS
	switch ops {
	case "windows":
		b, err := handleWindowsBrowserSearch()
		if err != nil {
			return "", err
		}
		return b, nil
	case "darwin":
		b, err := handleOSXBrowserSearch()
		if err != nil {
			return "", err
		}
		return b, nil

	case "linux":
		b, err := handleLinuxBrowserSearch()
		if err != nil {
			return "", err
		}
		return b, nil

	default:
		fmt.Printf("%s.\n", ops)
	}

	return "", nil
}

func HandleBrowserWizard(ctx *cli.Context) error {

	browserName, err := FindDefaultBrowser()

	if err != nil {
		return err
	}

	//if granted wasn't able to find the browser automatically request it from the user
	if browserName == "" {
		browserName, err = handleManualBrowserSelection()

		if err != nil {
			return err
		}
	}

	if strings.Contains(strings.ToLower(browserName), "chrome") ||
		strings.Contains(strings.ToLower(browserName), "brave") ||
		strings.Contains(strings.ToLower(browserName), "edge") {

		fmt.Fprintf(os.Stderr, "ℹ️  Granted has detected that your default browser is a Chromium based browser (Chrome, Brave, Edge)")

		alert := color.New(color.Bold, color.FgGreen).SprintFunc()
		fmt.Fprintf(os.Stderr, "\n%s\n", alert("✅  Granted will default to using Chromium profiles."))

		os.Exit(0)

	} else if strings.Contains(strings.ToLower(browserName), "firefox") {
		fmt.Fprintf(os.Stderr, "ℹ️  Granted has detected that your default browser is Mozilla Firefox.\n")
		fmt.Fprintf(os.Stderr, "ℹ️  You will need to download and install an extension for firefox to use Granted to its full potential\n")

		label := "\nTake me to download extension?\n"

		withStdio := survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)
		in := &survey.Confirm{
			Message: label,
			Default: true,
		}
		var confirm bool
		err := testable.AskOne(in, &confirm, withStdio)
		if err != nil {
			return err
		}

		if !confirm {
			return errors.New("cancelled browser wizard")
		}

		//TODO: replace this with a real marketplace link?
		//err = browser.OpenURL("https://drive.google.com/file/d/11zH06W9pzHmOgvdI5OiraMVBcL3AMpM-/view")
		//This was previously working in the old repo but now isnt?

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

		cmd := exec.Command(firefoxPath,
			"--new-tab",
			"https://drive.google.com/file/d/11zH06W9pzHmOgvdI5OiraMVBcL3AMpM-/view")
		err = cmd.Start()
		if err != nil {
			return err
		}

		// detach from this new process because it continues to run
		cmd.Process.Release()
		if err != nil {
			return err
		}
		time.Sleep(time.Second * 2)
		alert := color.New(color.Bold, color.FgGreen).SprintFunc()

		fmt.Fprintf(os.Stderr, "\n%s\n", alert("✅  Firefox set as default browser"))
		os.Exit(0)
	} else {
		fmt.Fprintf(os.Stderr, "ℹ️  Granted detected that you're using %s as your default browser", browserName)

		fmt.Fprintf(os.Stderr, "ℹ️  It is recommended to use Firefox or a chromium based browser to make use of all Granteds features")

		//do we want to ask them to change default browser here?

		os.Exit(0)
	}
	return nil
}

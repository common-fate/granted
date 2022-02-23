package browsers

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

//service mapping
var ServiceMap = map[string]string{
	"":               "console",
	"ec2":            "ec2/v2",
	"sso":            "singlesignon",
	"ecs":            "ecs",
	"eks":            "eks",
	"athena":         "athena",
	"cloudmap":       "cloudmap",
	"c9":             "cloud9",
	"cloudform":      "cloudformation",
	"cf":             "cloudfront",
	"ct":             "cloudtrail",
	"ddb":            "dynamodbv2",
	"ebs":            "elasticbeanstalk",
	"ecr":            "ecr",
	"grafana":        "grafana",
	"lambda":         "lambda",
	"route53":        "route53/v2",
	"s3":             "s3",
	"secretsmanager": "secretsmanager",
	"iam":            "iamv2",
}

// @TODO these file paths need to be verified
// alternatively, find a better way to get the exec path for a given browser
// @verified
const ChromePathMac = "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"

// @to-verify
const ChromePathLinux = `/usr/bin/google-chrome`

// @to-verify
const ChromePathWindows = `\Program Files\Google\Chrome\Application\chrome.exe`

// @verified
const BravePathMac = "/Applications/Brave Browser.app/Contents/MacOS/Brave Browser"

// @to-verify
const BravePathLinux = `/usr/bin/brave-browser`

// @verified
const BravePathWindows = `\Program Files\BraveSoftware\Brave-Browser\Application\brave.exe`

// @verified
const EdgePathMac = "/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge"

// @to-verify
const EdgePathLinux = `/usr/bin/edge`

// @verified
const EdgePathWindows = `\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`

func OpenWithChromiumProfile(url string, labels RoleLabels, selectedBrowser Browser) error {
	opSys := runtime.GOOS
	chromePath := ""
	switch selectedBrowser {
	case BrowserChrome:
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
	case BrowserBrave:
		switch opSys {
		case "windows":
			chromePath = BravePathWindows
		case "darwin":
			chromePath = BravePathMac
		case "linux":
			chromePath = BravePathLinux
		default:
			return errors.New("os not supported")
		}
	case BrowserEdge:
		switch opSys {
		case "windows":
			chromePath = EdgePathWindows
		case "darwin":
			chromePath = EdgePathMac
		case "linux":
			chromePath = EdgePathLinux
		default:
			return errors.New("os not supported")
		}
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

		userDataPath := path.Join(grantedFolder, "chromium-profiles", fmt.Sprintf("%v", selectedBrowser))

		//check the file path
		_, err = os.Stat(path.Join(userDataPath, labels.MakeExternalProfileTitle()))
		if err != nil {
			return err
		}

		cmd := exec.Command(chromePath,
			fmt.Sprintf("--user-data-dir=%s", userDataPath), "--profile-directory="+labels.MakeExternalProfileTitle(), "--no-first-run", "--no-default-browser-check", url,
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

const FirefoxPathWindows = `\Program Files\Mozilla Firefox\firefox.exe`

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

	tabURL := fmt.Sprintf("ext+granted-containers:name=%s&url=%s", labels.MakeExternalProfileTitle(), url.QueryEscape(urlString))
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
	Profile string
	Region  string
}

func (r *RoleLabels) MakeExternalProfileTitle() string {

	if r.Region != "" {
		return r.Profile + "(" + r.Region + ")"

	}
	return r.Profile
}

type Browser int

const (
	BrowerFirefox Browser = iota
	BrowserChrome
	BrowserBrave
	BrowserEdge
	BrowserDefault
)

func LaunchConsoleSession(sess Session, labels RoleLabels, service string, region string) error {
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

	dest, err := makeDestinationURL(service, region)

	if err != nil {
		return err
	}
	q = u.Query()
	q.Add("Action", "login")
	q.Add("Issuer", "")
	q.Add("Destination", dest)
	q.Add("SigninToken", token.SigninToken)
	u.RawQuery = q.Encode()
	cfg, _ := config.Load()
	if cfg == nil {
		return browser.OpenURL(u.String())
	}
	switch cfg.DefaultBrowser {
	case FirefoxKey:
		return OpenWithFirefoxContainer(u.String(), labels)
	case ChromeKey:
		return OpenWithChromiumProfile(u.String(), labels, BrowserChrome)
	case BraveKey:
		// @TODO use brave
		return OpenWithChromiumProfile(u.String(), labels, BrowserBrave)
	case EdgeKey:
		// @TODO use edge
		return OpenWithChromiumProfile(u.String(), labels, BrowserEdge)
	default:
		return browser.OpenURL(u.String())
	}
}

func makeDestinationURL(service string, region string) (string, error) {

	if region == "" {
		region = "us-east-1"
	}
	prefix := "https://console.aws.amazon.com/"

	serv := ServiceMap[service]
	if serv == "" {
		return "", fmt.Errorf("\nservice not found, please enter a valid service")
	}

	dest := prefix + serv + "/home"

	//NOTE here: excluding iam here and possibly others as the region isnt in the uri of the webpage on the console
	if region != "" || serv != "iam" {
		dest = dest + "?region=" + region
	}

	return dest, nil
}

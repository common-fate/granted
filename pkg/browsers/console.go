package browsers

import (
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/common-fate/granted/pkg/config"
	"github.com/fatih/color"
	"github.com/pkg/browser"
)

// ServiceMap maps CLI flags to AWS console URL paths.
// e.g. passing in `-r ec2` will open the console at the ec2/v2 URL.
var ServiceMap = map[string]string{
	"":               "console",
	"ec2":            "ec2/v2",
	"sso":            "singlesignon",
	"ecs":            "ecs",
	"eks":            "eks",
	"athena":         "athena",
	"cloudmap":       "cloudmap",
	"c9":             "cloud9",
	"cfn":            "cloudformation",
	"cloudformation": "cloudformation",
	"cloudwatch":     "cloudwatch",
	"gd":             "guardduty",
	"l":              "lambda",
	"cw":             "cloudwatch",
	"cf":             "cloudfront",
	"ct":             "cloudtrail",
	"ddb":            "dynamodbv2",
	"ebs":            "elasticbeanstalk",
	"ecr":            "ecr",
	"grafana":        "grafana",
	"lambda":         "lambda",
	"route53":        "route53/v2",
	"r53":            "route53/v2",
	"s3":             "s3",
	"secretsmanager": "secretsmanager",
	"iam":            "iamv2",
}

var globalServiceMap = map[string]bool{
	"iam":     true,
	"route53": true,
	"r53":     true,
}

func OpenWithChromiumProfile(url string, labels BrowserOpts, selectedBrowser Browser) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	chromePath := cfg.CustomBrowserPath
	if chromePath == "" {
		return fmt.Errorf("default browser not configured. run `granted browser set` to configure")
	}
	// check if the default chrome location is accessible
	_, err = os.Stat(chromePath)
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

		cmd := exec.Command(chromePath,
			fmt.Sprintf("--user-data-dir=%s", userDataPath), "--profile-directory="+labels.MakeExternalProfileTitle(), "--no-first-run", "--no-default-browser-check", url,
		)
		err = cmd.Start()
		if err != nil {
			fmt.Fprintf(color.Error, "\nGranted was unable to open a browser session automatically")
			//allow them to try open the url manually
			ManuallyOpenURL(url)
			return nil
		}
		// detach from this new process because it continues to run
		return cmd.Process.Release()
	}
	return errors.New("could not locate a Chrome installation")
}

func ManuallyOpenURL(url string) {
	alert := color.New(color.Bold, color.FgYellow).SprintFunc()
	fmt.Fprintf(os.Stdout, "\nOpen session manaually using the following url:\n")
	fmt.Fprintf(os.Stdout, "\n%s\n", alert("", url))
}

func MakeFirefoxContainerURL(urlString string, ops BrowserOpts) string {

	tabURL := fmt.Sprintf("ext+granted-containers:name=%s&url=%s", ops.MakeExternalFirefoxTitle(), url.QueryEscape(urlString))
	return tabURL
}

func OpenWithFirefoxContainer(urlString string, ops BrowserOpts) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	firefoxPath := cfg.CustomBrowserPath
	if firefoxPath == "" {
		return fmt.Errorf("default browser not configured. run `granted browser set` to configure")
	}

	tabURL := MakeFirefoxContainerURL(urlString, ops)
	cmd := exec.Command(firefoxPath,
		"--new-tab",
		tabURL)
	err = cmd.Start()
	if err != nil {
		fmt.Fprintf(color.Error, "\nGranted was unable to open a browser session automatically")
		//allow them to try open the url manually
		ManuallyOpenURL(tabURL)
		return nil
	}
	// detach from this new process because it continues to run
	return cmd.Process.Release()

}

type Session struct {
	SessionID    string `json:"sessionId"`
	SesssionKey  string `json:"sessionKey"`
	SessionToken string `json:"sessionToken"`
}

func SessionFromCredentials(creds aws.Credentials) Session {
	return Session{SessionID: creds.AccessKeyID, SesssionKey: creds.SecretAccessKey, SessionToken: creds.SessionToken}
}

type BrowserOpts struct {
	// the name of the role
	Profile string
	Region  string
	Service string
}

func (r *BrowserOpts) MakeExternalFirefoxTitle() string {
	if r.Region != "" {
		return r.Profile
	}
	return r.Profile
}

func (r *BrowserOpts) MakeExternalProfileTitle() string {
	n := r.Profile
	if r.Region != "" {
		n = r.Profile + "(" + r.Region + ")"

	}

	h := fnv.New32a()
	h.Write([]byte(n))

	hash := fmt.Sprint(h.Sum32())
	return hash

}

type Browser int

const (
	BrowerFirefox Browser = iota
	BrowserChrome
	BrowserBrave
	BrowserEdge
	BrowserChromium
	BrowserDefault
)

func MakeUrl(sess Session, opts BrowserOpts, service string, region string) (string, error) {
	sessJSON, err := json.Marshal(sess)
	if err != nil {
		return "", err
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
		return "", err
	}
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("opening console failed with code %v", res.StatusCode)
	}

	token := struct {
		SigninToken string `json:"SigninToken"`
	}{}

	err = json.NewDecoder(res.Body).Decode(&token)
	if err != nil {
		return "", err
	}

	u = url.URL{
		Scheme: "https",
		Host:   "signin.aws.amazon.com",
		Path:   "/federation",
	}

	dest, err := makeDestinationURL(service, region)

	if err != nil {
		return "", err
	}
	q = u.Query()
	q.Add("Action", "login")
	q.Add("Issuer", "")
	q.Add("Destination", dest)
	q.Add("SigninToken", token.SigninToken)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func LaunchConsoleSession(sess Session, opts BrowserOpts, service string, region string) error {
	url, err := MakeUrl(sess, opts, service, region)
	if err != nil {
		return err
	}

	cfg, _ := config.Load()
	if cfg == nil {
		return browser.OpenURL(url)
	}
	switch cfg.DefaultBrowser {
	case FirefoxKey:
		return OpenWithFirefoxContainer(url, opts)
	case ChromeKey:
		return OpenWithChromiumProfile(url, opts, BrowserChrome)
	case BraveKey:
		return OpenWithChromiumProfile(url, opts, BrowserBrave)
	case EdgeKey:
		return OpenWithChromiumProfile(url, opts, BrowserEdge)
	case ChromiumKey:
		return OpenWithChromiumProfile(url, opts, BrowserChromium)
	default:
		return browser.OpenURL(url)
	}
}

func makeDestinationURL(service string, region string) (string, error) {

	if region == "" {
		region = "us-east-1"
	}
	prefix := "https://console.aws.amazon.com/"

	serv := ServiceMap[service]
	if serv == "" {
		var validServices []string
		for s := range ServiceMap {
			validServices = append(validServices, s)
		}
		// present the strings in alphabetical order.
		// Yes, this is a bit of computation - but our arrays are quite small
		// and this avoids the need to keep the ServiceMap alphabetically sorted when developing Granted.
		sort.Strings(validServices)

		return "", fmt.Errorf("\nservice %s not found, please enter a valid service shortcut.\nValid service shortcuts: [%s]\n", service, strings.Join(validServices, ", "))

	}

	dest := prefix + serv + "/home"

	//excluding region here if the service is apart of the global service list
	//incomplete list of global services
	_, global := globalServiceMap[service]
	hasRegion := region != ""
	if !global && hasRegion {
		dest = dest + "?region=" + region

	}

	return dest, nil
}

func PromoteUseFlags(labels BrowserOpts) {
	var m []string

	if labels.Region == "" {
		m = append(m, "use -r to open a specific region")
	}

	if labels.Service == "" {
		m = append(m, "use -s to open a specific service")
	}

	if labels.Region == "" || labels.Service == "" {
		fmt.Fprintf(color.Error, "\nℹ️  %s (https://docs.commonfate.io/granted/usage/console)\n", strings.Join(m, " or "))

	}
}

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

func OpenWithChromiumProfile(url string, labels RoleLabels, selectedBrowser Browser) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	chromePath := cfg.CustomBrowserPath
	if chromePath == "" {
		switch selectedBrowser {
		case BrowserChrome:
			chromePath, err = ChromePath()
		case BrowserBrave:
			chromePath, err = BravePath()
		case BrowserEdge:
			chromePath, err = EdgePath()
		case BrowserChromium:
			chromePath, err = ChromiumPath()
		default:
			return errors.New("os not supported")
		}
		if err != nil {
			return err
		}
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
			fmt.Fprintf(os.Stderr, "\nGranted was unable to open a browser session automatically")
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

func MakeFirefoxContainerURL(urlString string, labels RoleLabels) string {

	tabURL := fmt.Sprintf("ext+granted-containers:name=%s&url=%s", labels.MakeExternalFirefoxTitle(), url.QueryEscape(urlString))
	return tabURL
}

func OpenWithFirefoxContainer(urlString string, labels RoleLabels) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	firefoxPath := cfg.CustomBrowserPath
	if firefoxPath == "" {
		firefoxPath, err = FirefoxPath()
		if err != nil {
			return err
		}
	}

	tabURL := MakeFirefoxContainerURL(ursString, labels)
	cmd := exec.Command(firefoxPath,
		"--new-tab",
		tabURL)
	err = cmd.Start()
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nGranted was unable to open a browser session automatically")
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

type RoleLabels struct {
	// the name of the role
	Profile string
	Region  string
	Service string
}

func (r *RoleLabels) MakeExternalFirefoxTitle() string {

	hash := r.MakeExternalProfileTitle()
	if r.Region != "" {
		return r.Profile + hash

	}
	return r.Profile
}

func (r *RoleLabels) MakeExternalProfileTitle() string {
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

func MakeUrl(sess Session, labels RoleLabels, service string, region string) (error, string) {
	sessJSON, err := json.Marshal(sess)
	if err != nil {
		return err, ""
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
		return err, ""
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("opening console failed with code %v", res.StatusCode), ""
	}

	token := struct {
		SigninToken string `json:"SigninToken"`
	}{}

	err = json.NewDecoder(res.Body).Decode(&token)
	if err != nil {
		return err, ""
	}

	u = url.URL{
		Scheme: "https",
		Host:   "signin.aws.amazon.com",
		Path:   "/federation",
	}

	dest, err := makeDestinationURL(service, region)

	if err != nil {
		return err, ""
	}
	q = u.Query()
	q.Add("Action", "login")
	q.Add("Issuer", "")
	q.Add("Destination", dest)
	q.Add("SigninToken", token.SigninToken)
	u.RawQuery = q.Encode()
	return nil, u.String()
}

func LaunchConsoleSession(sess Session, labels RoleLabels, service string, region string) error {
	err, url := MakeUrl(sess, labels, service, region)

	if err != nil {
		return err
	}

	cfg, _ := config.Load()
	if cfg == nil {
		return browser.OpenURL(url)
	}
	switch cfg.DefaultBrowser {
	case FirefoxKey:
		return OpenWithFirefoxContainer(url, labels)
	case ChromeKey:
		return OpenWithChromiumProfile(url, labels, BrowserChrome)
	case BraveKey:
		return OpenWithChromiumProfile(url, labels, BrowserBrave)
	case EdgeKey:
		return OpenWithChromiumProfile(url, labels, BrowserEdge)
	case ChromiumKey:
		return OpenWithChromiumProfile(url, labels, BrowserChromium)
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

	//NOTE here: excluding iam here and possibly others as the region isnt in the uri of the webpage on the console
	if region != "" || serv != "iam" {
		dest = dest + "?region=" + region
	}

	return dest, nil
}

func PromoteUseFlags(labels RoleLabels) {
	var m []string

	if labels.Region == "" {
		m = append(m, "use -r to open a specific region")
	}

	if labels.Service == "" {
		m = append(m, "use -s to open a specific service")
	}

	if labels.Region == "" || labels.Service == "" {
		fmt.Fprintf(os.Stderr, "\nℹ️  %s (https://docs.commonfate.io/granted/usage/console)\n", strings.Join(m, " or "))

	}
}

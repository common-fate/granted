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
	"github.com/common-fate/granted/pkg/debug"
	"github.com/fatih/color"
	"github.com/pkg/browser"
)

// ServiceMap maps CLI flags to AWS console URL paths.
// e.g. passing in `-r ec2` will open the console at the ec2/v2 URL.
var ServiceMap = map[string]string{
	"":               "console",
	"ec2":            "ec2/v2",
	"sso":            "singlesignon",
	"c9":             "cloud9",
	"cfn":            "cloudformation",
	"gd":             "guardduty",
	"l":              "lambda",
	"cw":             "cloudwatch",
	"cf":             "cloudfront",
	"ct":             "cloudtrail",
	"ddb":            "dynamodbv2",
	"eb":             "elasticbeanstalk",
	"ebs":            "elasticbeanstalk",
	"route53":        "route53/v2",
	"r53":            "route53/v2",
	"iam":            "iamv2",
	"waf":            "wafv2",
	"dms":            "dms/v2",
	"param":          "systems-manager/parameters",
	"redshift":       "redshiftv2",
	"ssm":            "systems-manager",	
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

type PartitionHost int

const (
	Default PartitionHost = iota
	Gov
	Cn
	ISO
	ISOB
)

func (p PartitionHost) String() string {
	switch p {
	case Default:
		return "aws"
	case Gov:
		return "aws-us-gov"
	case Cn:
		return "aws-cn"
	case ISO:
		return "aws-iso"
	case ISOB:
		return "aws-iso-b"
	}
	return "aws"
}

func (p PartitionHost) HostString() string {
	switch p {
	case Default:
		return "signin.aws.amazon.com"
	case Gov:
		return "signin.amazonaws-us-gov.com"
	case Cn:
		return "signin.amazonaws.cn"
	}
	// Note: we're not handling the ISO and ISOB cases, I don't think they are supported by a public AWS console
	return "signin.aws.amazon.com"
}

func (p PartitionHost) ConsoleHostString() string {
	switch p {
	case Default:
		return "https://console.aws.amazon.com/"
	case Gov:
		return "https://console.amazonaws-us-gov.com/"
	case Cn:
		return "https://console.amazonaws.cn/"
	}
	// Note: we're not handling the ISO and ISOB cases, I don't think they are supported by a public AWS console
	return "https://console.aws.amazon.com/"
}

func GetPartitionFromRegion(region string) PartitionHost {
	partition := strings.Split(region, "-")
	if partition[0] == "cn" {
		return PartitionHost(Cn)
	}
	if partition[1] == "iso" {
		return PartitionHost(ISO)
	}
	if partition[1] == "isob" {
		return PartitionHost(ISOB)
	}
	if partition[1] == "gov" {
		return PartitionHost(Gov)
	}
	return PartitionHost(Default)
}

func MakeUrl(sess Session, opts BrowserOpts, service string, region string) (string, error) {

	sessJSON, err := json.Marshal(sess)

	if err != nil {
		return "", err
	}

	partition := GetPartitionFromRegion(region)
	debug.Fprintf(debug.VerbosityDebug, color.Error, "Partition is detected as %s for region %s...\n", partition, region)

	u := url.URL{
		Scheme: "https",
		Host:   partition.HostString(),
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
		Host:   partition.HostString(),
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

func OpenUrlWithCustomBrowser(url string) error {

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.CustomSSOBrowserPath != "" {
		cmd := exec.Command(cfg.CustomSSOBrowserPath, fmt.Sprintf(" %s ", url))
		err := cmd.Start()
		if err != nil {
			return err
		}
		// detach from this new process because it continues to run
		return cmd.Process.Release()
	}
	return nil

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
	partition := GetPartitionFromRegion(region)
	prefix := partition.ConsoleHostString()

	serv := ServiceMap[service]
	if serv == "" {
		color.New(color.FgYellow).Fprintf(color.Error, "[warning] we don't recognize service %s but we'll try and open it anyway (you may receive a 404 page)", service)
		serv = service
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

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
	"strconv"
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
	"eb":             "elasticbeanstalk",
	"ebs":            "elasticbeanstalk",
	"ecr":            "ecr",
	"grafana":        "grafana",
	"lambda":         "lambda",
	"route53":        "route53/v2",
	"r53":            "route53/v2",
	"s3":             "s3",
	"secretsmanager": "secretsmanager",
	"iam":            "iamv2",
	"waf":            "wafv2",
	"rds":            "rds",
	"dms":            "dms/v2",
	"mwaa":           "mwaa",
	"param":          "systems-manager/parameters",
	"redshift":       "redshiftv2",
	"sagemaker":      "sagemaker",
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

	region, err = expandRegion(region)
	if err != nil {
		return "", fmt.Errorf("couldn't parse region %s: %v", region, err)
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

func expandRegion(region string) (string, error) {
	// Region could come in one of three formats:
	// 1. No region specified
	if region == "" {
		return "us-east-1", nil
	}
	// 2. A fully-qualified region. Assume that if there's one dash, it's valid.
	if strings.Contains(region, "-") {
		return region, nil
	}
	var major, minor, num string
	idx := 1 // Number of characters consumed from region
	// 3. Otherwise, we have a shortened region, like ue1
	if len(region) < 2 {
		return "", fmt.Errorf("region too short, needs at least two characters (eg ue)")
	}
	// Region might be one or two letters
	switch region[0] {
	case 'u':
		{
			major = "us"
			if region[1] == 'g' {
				major = "us-gov"
				idx += 1
			} else if region[1] == 's' {
				// This will break if us-southeast-1 is ever created
				idx += 1
			}
		}
	case 'e':
		{
			major = "eu"
		}
	case 'a':
		{
			major = "ap"
			if region[1] == 'f' {
				major = "af"
				idx += 1
			} else if region[1] == 'p' {
				idx += 1
			}
		}
	case 'c':
		{
			major = "ca"
			if region[1] == 'n' {
				major = "cn"
				idx += 1
			} else if region[1] == 'a' {
				idx += 1
			}
		}
	case 'm':
		{
			major = "me"
			// This will break if me-east-1 is ever created
			if region[1] == 'e' {
				idx += 1
			}
		}
	case 's':
		{
			major = "sa"
			if region[1] == 'a' {
				idx += 1
			}
		}
	default:
		{
			return "", fmt.Errorf("unknown region major (hint: try using the first letter of the region)")
		}
	}
	region = region[idx:]
	idx = 1
	// Location might be one or two letters (n, nw)
	switch region[0] {
	case 'n', 's':
		{
			if region[0] == 'n' {
				minor = "north"
			} else {
				minor = "south"
			}
			if len(region) > 1 {
				if region[1] == 'w' {
					minor += "west"
					idx += 1

				} else if region[1] == 'e' {
					minor += "east"
					idx += 1
				}
			}
		}
	case 'e':
		{
			minor = "east"
		}
	case 'w':
		{
			minor = "west"
		}
	case 'c':
		{
			minor = "central"
		}
	default:
		{
			return "", fmt.Errorf("unknown region minor in %s (found major: %s)", region, major)
		}
	}
	region = region[idx:]
	if len(region) > 0 {
		_, err := strconv.Atoi(region)
		if err != nil {
			return "", fmt.Errorf("unknown region number in %s (found major: %s, minor: %s)", region, major, minor)
		}
		num = region
	} else {
		num = "1"
	}

	return fmt.Sprintf("%s-%s-%s", major, minor, num), nil
}

func makeDestinationURL(service string, region string) (string, error) {
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

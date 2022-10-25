package cfaws

import (
	"fmt"
	"net/url"
	"regexp"

	"github.com/bigkevmcd/go-configparser"
	"github.com/common-fate/clio"
	grantedConfig "github.com/common-fate/granted/pkg/config"
)

// GetGrantedApprovalsURL returns the URL which users can request access to a particular role at.
//
// To return a request URL, a base URL for a Granted Approvals deployment must be set. The base URL can be provided in
// a couple of ways and is read in the following order of priority:
//
// 1. By setting the '--url' flag with the Granted credentials_process command
//
// 2. By setting a global request URL with the command 'granted settings request-url set'
//
// If neither of the approaches above returns a URL, this method returns a message indicating that the request URL
// hasn't been set up.
func FormatAWSErrorWithGrantedApprovalsURL(awsError error, rawConfig configparser.Dict, gConf grantedConfig.Config, SSORoleName string, SSOAccountId string) error {
	cliErr := &clio.CLIError{
		Err: awsError.Error(),
	}
	// try and extract a --url flag from the AWS profile, like the following:
	//	[profile my-profile]
	//	credential_process = granted credential-process --url https://example.com
	// This flag takes the highest precendence if it is set.
	url := parseURLFlagFromConfig(rawConfig)
	if url == "" {
		// if the --url flag wasn't found, try and load the global request URL setting.
		url = gConf.AccessRequestURL
	}

	if url != "" {
		// if we have a request URL, we can prompt the user to make a request by visiting the URL.
		requestURL := buildRequestURL(url, SSORoleName, SSOAccountId)
		// need to escape the % symbol in the request url which has been query escaped so that fmt does';t try to substitute it
		cliErr.Messages = append(cliErr.Messages, clio.WarnMsg("You need to request access to this role:"), clio.WarnlnMsg(requestURL))
		return cliErr
	}

	// otherwise, there is no request URL configured. Let the user know that they can set one up if they are using Granted Approvals
	// remember that not all users of credential process will be using approvals
	cliErr.Messages = append(cliErr.Messages,
		clio.InfoMsg("It looks like you don't have the right permissions to access this role"),
		clio.InfoMsg("If you are using Granted Approvals to manage this role you can configure the Granted CLI with a request URL so that you can be directed to your Granted Approvals instance to make a new access request the next time you have this error"),
		clio.InfoMsg("To configure a URL to request access to this role with 'granted settings request-url set <YOUR_GRANTED_APPROVALS_URL'"),
	)
	return cliErr
}

// parseURLFlagFromConfig tries to extract the '--url' argument from the granted credentials_process command in an AWS profile.
// If the AWS profile looks like this:
//
//	[profile my-profile]
//	credential_process = granted credential-process --url https://example.com
//
// it will return 'https://example.com'. Otherwise, it returns an empty string
func parseURLFlagFromConfig(rawConfig configparser.Dict) string {
	credProcess, ok := rawConfig["credential_process"]
	if !ok {
		return ""
	}
	grantedRegex := regexp.MustCompile(`granted\s+credential-process`)
	hasGrantedCommand := grantedRegex.MatchString(credProcess)
	if !hasGrantedCommand {
		return ""
	}
	re := regexp.MustCompile(`--url\s+(\S+)`)
	matchedValues := re.FindStringSubmatch(credProcess)
	if len(matchedValues) > 1 {
		return matchedValues[1]
	}
	return ""
}

func buildRequestURL(grantedUrl string, SSORoleName string, SSOAccountId string) string {
	u, err := url.Parse(grantedUrl)
	if err != nil {
		return fmt.Sprintf("error building access request URL: %s", err.Error())
	}
	u.Path = "access"
	q := u.Query()
	q.Add("type", "commonfate/aws-sso")
	q.Add("permissionSetArn.label", SSORoleName)
	q.Add("accountId", SSOAccountId)
	u.RawQuery = q.Encode()

	return u.String()
}

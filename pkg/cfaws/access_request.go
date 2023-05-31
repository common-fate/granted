package cfaws

import (
	"regexp"

	"github.com/common-fate/clio"
	"github.com/common-fate/clio/clierr"
	"github.com/common-fate/granted/pkg/accessrequest"
	grantedConfig "github.com/common-fate/granted/pkg/config"
	"gopkg.in/ini.v1"
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
func FormatAWSErrorWithGrantedApprovalsURL(awsError error, rawConfig *ini.Section, gConf grantedConfig.Config, SSORoleName string, SSOAccountId string) error {
	cliError := &clierr.Err{
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
		latestRole := accessrequest.Role{
			Account: SSOAccountId,
			Role:    SSORoleName,
		}
		err := latestRole.Save()
		if err != nil {
			clio.Errorw("error saving latest role", "error", err)
		}

		// if we have a request URL, we can prompt the user to make a request by visiting the URL.
		requestURL := latestRole.URL(url)
		// need to escape the % symbol in the request url which has been query escaped so that fmt doesn't try to substitute it
		cliError.Messages = append(cliError.Messages, clierr.Warn("You need to request access to this role:"), clierr.Warn(requestURL), clierr.Warn("or run: 'granted exp request latest'"))
		return cliError
	}

	// otherwise, there is no request URL configured. Let the user know that they can set one up if they are using Granted Approvals
	// remember that not all users of credential process will be using approvals
	cliError.Messages = append(cliError.Messages,
		clierr.Info("It looks like you don't have the right permissions to access this role"),
		clierr.Info("If you are using Common Fate to manage this role you can configure the Granted CLI with a request URL so that you can be directed to your Granted Approvals instance to make a new access request the next time you have this error"),
		clierr.Info("To configure a URL to request access to this role with 'granted settings request-url set <YOUR_GRANTED_APPROVALS_URL'"),
	)
	return cliError
}

// parseURLFlagFromConfig tries to extract the '--url' argument from the granted credentials_process command in an AWS profile.
// If the AWS profile looks like this:
//
//	[profile my-profile]
//	credential_process = granted credential-process --url https://example.com
//
// it will return 'https://example.com'. Otherwise, it returns an empty string
func parseURLFlagFromConfig(rawConfig *ini.Section) string {
	credProcess, err := rawConfig.GetKey("credential_process")
	if err != nil {
		clio.Debug(err.Error())
		return ""
	}
	grantedRegex := regexp.MustCompile(`granted\s+credential-process`)
	hasGrantedCommand := grantedRegex.MatchString(credProcess.Value())
	if !hasGrantedCommand {
		return ""
	}
	re := regexp.MustCompile(`--url\s+(\S+)`)
	matchedValues := re.FindStringSubmatch(credProcess.Value())
	if len(matchedValues) > 1 {
		return matchedValues[1]
	}
	return ""
}

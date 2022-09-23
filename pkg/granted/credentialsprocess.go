package granted

import (
	"encoding/json"
	"fmt"
	urlLib "net/url"
	"os"
	"time"

	grantedConfig "github.com/common-fate/granted/pkg/config"
	"github.com/pkg/errors"

	"github.com/aws/smithy-go"
	"github.com/common-fate/granted/pkg/cfaws"
	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
)

/*
* AWS Creds consumed by credential_process must adhere to this schema
* https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-sourcing-external.html
 */
type AWSCredsStdOut struct {
	Version         int    `json:"Version"`
	AccessKeyID     string `json:"AccessKeyId"`
	SecretAccessKey string `json:"SecretAccessKey"`
	SessionToken    string `json:"SessionToken"`
	Expiration      string `json:"Expiration"`
}

var CredentialsProcess = cli.Command{
	Name:  "credential-process",
	Usage: "Exports AWS session credentials for use with AWS CLI credential_process",
	Flags: []cli.Flag{&cli.StringFlag{Name: "profile", Required: true}, &cli.StringFlag{Name: "url"}},
	Action: func(c *cli.Context) error {

		url := c.String("url")
		profileName := c.String("profile")

		if url == "" {
			gConf, err := grantedConfig.Load()
			if err != nil {
				return err
			}
			url = gConf.AccessRequestURL
		}

		profiles, err := cfaws.LoadProfiles()
		if err != nil {
			return err
		}

		profile, err := profiles.LoadInitialisedProfile(c.Context, profileName)
		if err != nil {
			return err
		}

		creds, err := profile.AssumeTerminal(c.Context, cfaws.ConfigOpts{Duration: time.Hour})
		if err != nil {
			// print the error so the user knows what went wrong.
			fmt.Fprintln(os.Stderr, err)

			serr, ok := err.(*smithy.OperationError)
			if ok && serr.ServiceID == "SSO" {
				// The user may be able to request access to the role if
				// they are using Granted Approvals.
				// Display an error message with the request URL, or a prompt
				// to set up the request URL if it's empty.
				fmt.Fprintln(os.Stderr, getGrantedApprovalsURL(url, profile))
			}

			// exit with an error status, as we haven't been able to assume the role.
			os.Exit(1)
		}

		var out AWSCredsStdOut
		out.AccessKeyID = creds.AccessKeyID
		out.Expiration = creds.Expires.Format(time.RFC3339)
		out.SecretAccessKey = creds.SecretAccessKey
		out.SessionToken = creds.SessionToken
		out.Version = 1

		jsonOut, err := json.Marshal(out)
		if err != nil {
			return errors.Wrap(err, "marshalling session credentials")
		}

		fmt.Println(string(jsonOut))
		return nil
	},
}

func getGrantedApprovalsURL(url string, profile *cfaws.Profile) string {
	if url == "" {
		return "If you're using Granted Approvals, set up a URL to request access to this role with 'granted settings request-url set <Your Granted Approvals URL>'"
	}
	u, err := urlLib.Parse(url)
	if err != nil {
		return fmt.Sprintf("error building access request URL: %s", err.Error())
	}
	u.Path = "access"
	q := u.Query()
	q.Add("type", "commonfate/aws-sso")
	q.Add("permissionSetArn.label", profile.AWSConfig.SSORoleName)
	q.Add("accountId", profile.AWSConfig.SSOAccountID)
	u.RawQuery = q.Encode()

	msg := color.YellowString("You need to request access to this role:\n" + u.String())

	return msg
}

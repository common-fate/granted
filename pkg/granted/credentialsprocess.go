package granted

import (
	"encoding/json"
	"fmt"
	"log"
	urlLib "net/url"
	"time"

	grantedConfig "github.com/common-fate/granted/pkg/config"

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

		profiles, err := cfaws.LoadProfiles()
		if err != nil {
			log.Fatal(err)
		}

		profile, err := profiles.LoadInitialisedProfile(c.Context, profileName)
		if err != nil {
			_, ok := err.(*smithy.OperationError)
			if ok {
				// if it failed to load initialised try load it from the config
				pr, loadErr := profiles.Profile(profileName)
				if loadErr != nil {
					log.Fatal(loadErr)
				}
				log.Fatal(getGrantedApprovalsURL(url, pr))
			} else {
				log.Fatalf("granted credential_process error for profile '%s' with err: %s", profileName, err.Error())
			}
		}

		creds, err := profile.AssumeTerminal(c.Context, cfaws.ConfigOpts{Duration: time.Hour})
		if err != nil {
			serr, ok := err.(*smithy.OperationError)
			if ok {
				// Prompt Granted-Approvals AR request
				if serr.ServiceID == "SSO" {
					log.Fatal(getGrantedApprovalsURL(url, profile))
				}
			} else {
				log.Fatalln("\nError running credential with profile: "+profileName, err.Error())
			}
		}

		var out AWSCredsStdOut
		out.AccessKeyID = creds.AccessKeyID
		out.Expiration = creds.Expires.Format(time.RFC3339)
		out.SecretAccessKey = creds.SecretAccessKey
		out.SessionToken = creds.SessionToken
		out.Version = 1

		jsonOut, err := json.Marshal(out)
		if err != nil {
			log.Fatalln("\nUnhandled error when unmarshalling json creds")
		}
		fmt.Println(string(jsonOut))

		return nil
	},
}

func getGrantedApprovalsURL(url string, profile *cfaws.Profile) string {
	if url == "" {
		gConf, err := grantedConfig.Load()
		if err != nil {
			log.Fatal("no url passed, failed to load Config for GrantedApprovalsURL")
		}
		if gConf.GrantedApprovalsURL == "" {
			log.Fatal("\nIf you're using Granted Approvals, set up a URL to request access to this role with 'granted settings request-url set <Your Granted Approvals URL>'")
		} else {
			url = gConf.GrantedApprovalsURL
		}
	}
	u, err := urlLib.Parse(url)
	if err != nil {
		log.Fatal("Error when generating Granted Approvals URL: ", err)
	}
	u.Path = "access"
	q := u.Query()
	q.Add("type", "commonfate/aws-sso") // hardcoded to begin with (only supporting aws sso)
	q.Add("permissionSetArn.label", profile.AWSConfig.SSORoleName)
	q.Add("accountId", profile.AWSConfig.SSOAccountID)
	u.RawQuery = q.Encode()

	requestMsg := color.YellowString("\n\nYou need to request access to this role:\n" + u.String() + "\n")

	return requestMsg
}

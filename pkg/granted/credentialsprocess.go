package granted

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

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
			url = "http://localhost:3000/"
		}
		// In future we will support manual storing of url in config...
		// conf, err := config.Load()
		// if err != nil {
		// 	log.Fatal("Failed to load Config for GrantedApprovalsUrl")
		// }
		// url = conf.GrantedApprovalsUrl
		if url == "" {
			log.Fatal("It looks like you haven't setup your GrantedApprovalsUrl\nTo do so please run: " + c.App.Name + " setup")
		}

		profiles, err := cfaws.LoadProfiles()
		if err != nil {
			log.Fatal(err)
		}

		profile, err := profiles.LoadInitialisedProfile(c.Context, profileName)
		if err != nil {
			_, ok := err.(*smithy.OperationError)
			if ok {
				// if it failed to load initialised try load it from the config
				pr, err := profiles.Profile(profileName)
				if err != nil {
					log.Fatal(err)
				}
				log.Fatal(getGrantedApprovalsUrl(url, pr))
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
					log.Fatal(getGrantedApprovalsUrl(url, profile))
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

func getGrantedApprovalsUrl(url string, profile *cfaws.Profile) string {
	providerType := "commonfate%2Faws-sso"
	return color.YellowString("\n\nYou need to request access to this role:"+"\n%saccess?type=%s&roleName=%s&accountId=%s\n", url, providerType, profile.AWSConfig.SSORoleName, profile.AWSConfig.SSOAccountID)
}

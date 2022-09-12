package granted

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/aws/smithy-go"
	"github.com/common-fate/granted/pkg/cfaws"
	"github.com/common-fate/granted/pkg/config"
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
	Name:        "credentialsprocess",
	Usage:       "",
	Hidden:      true,
	Subcommands: []*cli.Command{&ConfigSetup},
	Flags:       []cli.Flag{&cli.StringFlag{Name: "profile"}, &cli.StringFlag{Name: "rule"}},
	Action: func(c *cli.Context) error {

		url := ""
		profileName := c.String("profile")
		// ruleId := c.String("rule")

		conf, err := config.Load()
		if err != nil {
			log.Fatal("Failed to load Config for GrantedApprovalsUrl")
		}
		url = conf.GrantedApprovalsUrl
		if url == "" {
			log.Fatal("It looks like you haven't setup your GrantedApprovalsUrl\nTo do so please run: " + c.App.Name + " setup")
		}

		if profileName == "" {
			log.Fatalln("Mis-configured aws config file.\n--profile flag must be passed")
		}

		var profile *cfaws.Profile

		profiles, err := cfaws.LoadProfiles()
		if err != nil {
			return err
		}

		if profiles.HasProfile(profileName) {
			profile, err = profiles.LoadInitialisedProfile(c.Context, profileName)
			if err != nil {
				return err
			}

			creds, err := profile.AssumeTerminal(c.Context, cfaws.ConfigOpts{Duration: time.Hour})

			if err != nil {
				serr, ok := err.(*smithy.OperationError)
				if ok {
					// @TODO: this still seems to trigger when ...
					// 1. revoke any access to cf-dev
					// 2. request access and wait for grant = active
					// 3. try run `aws s3 ls --profile granted.cf-dev`
					// 4. it should auto assume the role, but instead it throws below error \/ \/
					if serr.ServiceID == "SSO" {

						// type=aws-sso&roleName=AWSAdministratorAccess&accountId=123456789012
						credsType := "aws-sso"
						roleName := "AWSAdministratorAccess"
						accountId := "123456789012"

						roleUrl := fmt.Sprintf("%srequest?type=%s&roleName=%s&accountId=%s", url, credsType, roleName, accountId)

						// Guide user to common fate if error
						s := fmt.Sprintf(color.YellowString("\n\nYou need to request access to this role:")+"\n%s", roleUrl)

						log.Fatal(s)
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
			fmt.Print(string(jsonOut))
		}

		return nil
	},
}

// @TODO: we may also want to add an automated process that handles sync to ensure the config is never stale

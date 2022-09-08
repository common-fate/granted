package granted

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
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
	Name:        "credentialsprocess",
	Usage:       "",
	Hidden:      true,
	Subcommands: []*cli.Command{&ConfigSetup},
	Flags:       []cli.Flag{&cli.StringFlag{Name: "profile"}},
	Action: func(c *cli.Context) error {

		profileName := c.String("profile")

		if profileName == "" {
			log.Fatalln("Mis-configured aws config file.\n--profile flag must be passed")
		}

		// Check if the session can be assumed
		err := CheckIfRequiresApproval(c, profileName)
		if err != nil {
			os.Exit(1)
			fmt.Fprintln(color.Error, "Unhandled exception while initializing SSO")
		}
		// Yes it (now) can be assumed, run standard `aws aws-sso-credential-process..`
		// out, err := exec.Command("aws-sso-credential-process", "credential-process", "--profile", profileName).Output()
		out, err := exec.Command("aws-sso-credential-process", "credential-process", "--profile", profileName).Output()
		if err != nil {
			log.Fatalln("Error running native aws-sso-credential-process with profile: "+profileName, err.Error())
			log.Fatal(err)
		}
		// Export the credentials in json format
		fmt.Print(string(out))

		return nil
	},
}

/**
* IO = return nil, log fatal error if User does not have access to the SSO role
* IO = return error, log nothing if unhandled exception
 */
func CheckIfRequiresApproval(c *cli.Context, profileName string) error {

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

		_, err := profile.AssumeTerminal(c.Context, cfaws.ConfigOpts{Duration: time.Hour})

		if err != nil {
			serr, ok := err.(*smithy.OperationError)
			if ok {
				if serr.ServiceID == "SSO" {
					baseUrl, ruleId := "internal.prod.granted.run/", "rul_2BtW97o6jTacUuzxNJZorACn5v0"
					// Guide user to common fate if error
					s := fmt.Sprintf(color.YellowString("\n\nYou need to request access to this role:")+"\nhttps://%s/access/request/%s\n", baseUrl, ruleId)

					log.Fatal(s)
				}
			}
			return err
		}
		return nil
	}
	return errors.New("unhandled error, could not assume profile")
}

// @TODO: we may also want to add an automated process that handles sync to ensure the config is never stale

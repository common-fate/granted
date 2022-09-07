package granted

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/AlecAivazis/survey/v2"
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

		// Check if the session can be assumed
		err := CheckIfRequiresApproval(c, "cf-dev")
		if err != nil {
			fmt.Fprintln(color.Error, "Unhandled exception while initializing SSO")
			os.Exit(1)
		}
		// Yes it can be assumed, run standard `aws aws-sso-credential-process..`
		// err = exec.Command("aws-sso-credential-process", "--profile", "cf-dev").Run()

		// Export the credentials in json format

		// Else log a message to the console

		return nil
	},
}

var ConfigSetup = cli.Command{
	Name:  "setup",
	Usage: "Alters your AWS Config credential_processs",
	Action: func(c *cli.Context) error {

		fmt.Println("This will override the `credential_process` in your config file.")
		withStdio := survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)
		in := &survey.Confirm{
			Message: "Would you like to proceed?",
			Default: false,
		}
		var confirm bool
		err := survey.AskOne(in, &confirm, withStdio)
		if err != nil {
			return err
		}
		if !confirm {
			return errors.New("cancelled alias installation")
		}

		// load the default config

		// Itterate over each section

		// If a section contains a non-standard credential_process

		// Prompt the user for confrimation

		// Write to credential_process our custom script

		// Done :)

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
					baseUrl, ruleId := "granted.dev", "rul_2BtW97o6jTacUuzxNJZorACn5v0"
					// Guide user to common fate if error
					s := fmt.Sprintf("ERROR: You need to request access to this role: https://%s/access/request/%s", baseUrl, ruleId)

					fmt.Println(s)
					os.Exit(1)
				}
			}
			return err
		}
		return nil
	}
	return errors.New("unhandled error, could not assume profile")
}

// @TODO: we may also want to add an automated process that handles sync to ensure the config is never stale

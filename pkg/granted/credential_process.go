package granted

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/pkg/errors"

	"github.com/common-fate/granted/pkg/cfaws"
	"github.com/urfave/cli/v2"
)

// AWS Creds consumed by credential_process must adhere to this schema
// https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-sourcing-external.html
type awsCredsStdOut struct {
	Version         int    `json:"Version"`
	AccessKeyID     string `json:"AccessKeyId"`
	SecretAccessKey string `json:"SecretAccessKey"`
	SessionToken    string `json:"SessionToken"`
	Expiration      string `json:"Expiration"`
}

var CredentialProcess = cli.Command{
	Name:  "credential-process",
	Usage: "Exports AWS session credentials for use with AWS CLI credential_process",
	Flags: []cli.Flag{&cli.StringFlag{Name: "profile", Required: true}, &cli.StringFlag{Name: "url"}},
	Action: func(c *cli.Context) error {

		profileName := c.String("profile")
		profiles, err := cfaws.LoadProfiles()
		if err != nil {
			return err
		}

		profile, err := profiles.LoadInitialisedProfile(c.Context, profileName)
		if err != nil {
			return err
		}

		duration := time.Hour
		if profile.AWSConfig.RoleDurationSeconds != nil {
			duration = *profile.AWSConfig.RoleDurationSeconds
		}

		creds, err := profile.AssumeTerminal(c.Context, cfaws.ConfigOpts{Duration: duration})
		if err != nil {
			// print the error so the user knows what went wrong.
			fmt.Fprintln(os.Stderr, err)

			// exit with an error status, as we haven't been able to assume the role.
			os.Exit(1)
		}

		out := awsCredsStdOut{
			Version:         1,
			AccessKeyID:     creds.AccessKeyID,
			SecretAccessKey: creds.SecretAccessKey,
			SessionToken:    creds.SessionToken,
			Expiration:      creds.Expires.Format(time.RFC3339),
		}

		jsonOut, err := json.Marshal(out)
		if err != nil {
			return errors.Wrap(err, "marshalling session credentials")
		}

		fmt.Println(string(jsonOut))
		return nil
	},
}

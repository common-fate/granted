package granted

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/common-fate/granted/pkg/cfaws"
	"github.com/common-fate/granted/pkg/securestorage"
	"github.com/urfave/cli/v2"
)

// AWS Creds consumed by credential_process must adhere to this schema
// https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-sourcing-external.html
type awsCredsStdOut struct {
	Version         int    `json:"Version"`
	AccessKeyID     string `json:"AccessKeyId"`
	SecretAccessKey string `json:"SecretAccessKey"`
	SessionToken    string `json:"SessionToken,omitempty"`
	Expiration      string `json:"Expiration,omitempty"`
}

var CredentialProcess = cli.Command{
	Name:  "credential-process",
	Usage: "Exports AWS session credentials for use with AWS CLI credential_process",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "profile", Required: true},
		&cli.StringFlag{Name: "url"},
		&cli.DurationFlag{Name: "window", Value: 15 * time.Minute},
	},
	Action: func(c *cli.Context) error {

		profileName := c.String("profile")

		secureSessionCredentialStorage := securestorage.NewSecureSessionCredentialStorage()
		creds, ok, err := secureSessionCredentialStorage.GetCredentials(profileName)
		if err != nil {
			return err
		}

		if !ok || (creds.CanExpire && time.Now().After(creds.Expires.Add(-c.Duration("window")))) {
			profiles, err := cfaws.LoadProfilesFromDefaultFiles()
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

			creds, err = profile.AssumeTerminal(c.Context, cfaws.ConfigOpts{Duration: duration, UsingCredentialProcess: true})
			if err != nil {
				return err
			}

			if err := secureSessionCredentialStorage.StoreCredentials(profileName, creds); err != nil {
				return err
			}
		}

		out := awsCredsStdOut{
			Version:         1,
			AccessKeyID:     creds.AccessKeyID,
			SecretAccessKey: creds.SecretAccessKey,
			SessionToken:    creds.SessionToken,
		}
		if creds.CanExpire {
			out.Expiration = creds.Expires.Format(time.RFC3339)
		}

		jsonOut, err := json.Marshal(out)
		if err != nil {
			return errors.Wrap(err, "marshalling session credentials")
		}

		fmt.Println(string(jsonOut))
		return nil
	},
}

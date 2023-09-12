package granted

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/pkg/errors"

	"github.com/common-fate/clio"
	"github.com/common-fate/granted/pkg/cfaws"
	"github.com/common-fate/granted/pkg/config"
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
		&cli.BoolFlag{Name: "auto-login", Usage: "automatically open the configured browser to log in if needed"},
	},
	Action: func(c *cli.Context) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		var needsRefresh bool
		var credentials aws.Credentials
		profileName := c.String("profile")
		autoLogin := c.Bool("auto-login")
		secureSessionCredentialStorage := securestorage.NewSecureSessionCredentialStorage()
		clio.Debugw("running credential process with config", "profile", profileName, "url", c.String("url"), "window", c.Duration("window"), "disableCredentialProcessCache", cfg.DisableCredentialProcessCache)
		if !cfg.DisableCredentialProcessCache {
			creds, ok, err := secureSessionCredentialStorage.GetCredentials(profileName)
			if err != nil {
				return err
			}
			if !ok {
				clio.Debugw("refreshing credentials", "reason", "not found")
				needsRefresh = true
			} else {
				clio.Debugw("credentials found in cache", "expires", creds.Expires.String(), "canExpire", creds.CanExpire, "timeNow", time.Now().String(), "refreshIfBeforeNow", creds.Expires.Add(-c.Duration("window")).String())
				if creds.CanExpire && creds.Expires.Add(-c.Duration("window")).Before(time.Now()) {
					clio.Debugw("refreshing credentials", "reason", "credentials are expired")
					needsRefresh = true
				} else {
					clio.Debugw("using cached credentials")
					credentials = creds
				}
			}
		} else {
			clio.Debugw("refreshing credentials", "reason", "credential process cache is disabled via config")
			needsRefresh = true
		}

		if needsRefresh {
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

			credentials, err = profile.AssumeTerminal(c.Context, cfaws.ConfigOpts{Duration: duration, UsingCredentialProcess: true, CredentialProcessAutoLogin: autoLogin})
			if err != nil {
				return err
			}
			if !cfg.DisableCredentialProcessCache {
				clio.Debugw("storing refreshed credentials in credential process cache", "expires", credentials.Expires.String(), "canExpire", credentials.CanExpire, "timeNow", time.Now().String())
				if err := secureSessionCredentialStorage.StoreCredentials(profileName, credentials); err != nil {
					return err
				}
			}
		}

		out := awsCredsStdOut{
			Version:         1,
			AccessKeyID:     credentials.AccessKeyID,
			SecretAccessKey: credentials.SecretAccessKey,
			SessionToken:    credentials.SessionToken,
		}
		if credentials.CanExpire {
			out.Expiration = credentials.Expires.Format(time.RFC3339)
		}

		jsonOut, err := json.Marshal(out)
		if err != nil {
			return errors.Wrap(err, "marshalling session credentials")
		}

		fmt.Println(string(jsonOut))
		return nil
	},
}

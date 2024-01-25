package granted

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/pkg/errors"

	"github.com/common-fate/clio"
	"github.com/common-fate/granted/pkg/cfaws"
	"github.com/common-fate/granted/pkg/config"
	"github.com/common-fate/granted/pkg/securestorage"
	"github.com/urfave/cli/v2"

	accessv1alpha1 "github.com/common-fate/sdk/gen/commonfate/access/v1alpha1"
	entityv1alpha1 "github.com/common-fate/sdk/gen/commonfate/entity/v1alpha1"
	"github.com/common-fate/sdk/service/access"

	sdkconfig "github.com/common-fate/sdk/config"
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
		ctx := c.Context

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		profileName := c.String("profile")
		autoLogin := c.Bool("auto-login")
		secureSessionCredentialStorage := securestorage.NewSecureSessionCredentialStorage()
		clio.Debugw("running credential process with config", "profile", profileName, "url", c.String("url"), "window", c.Duration("window"), "disableCredentialProcessCache", cfg.DisableCredentialProcessCache)

		useCache := !cfg.DisableCredentialProcessCache

		if useCache {
			// try and look up session credentials from the secure storage cache.
			cachedCreds, err := secureSessionCredentialStorage.GetCredentials(profileName)
			if err != nil {
				clio.Debugw("error loading cached credentials", "error", err)
			} else if cachedCreds == nil {
				clio.Debugw("refreshing credentials", "reason", "cachedCreds was nil")
			} else if cachedCreds.CanExpire && cachedCreds.Expires.Add(-c.Duration("window")).Before(time.Now()) {
				clio.Debugw("refreshing credentials", "reason", "credentials are expired")
			} else {
				// if we get here, the cached session credentials are valid
				clio.Debugw("credentials found in cache", "expires", cachedCreds.Expires.String(), "canExpire", cachedCreds.CanExpire, "timeNow", time.Now().String(), "refreshIfBeforeNow", cachedCreds.Expires.Add(-c.Duration("window")).String())
				return printCredentials(*cachedCreds)
			}
		}

		if !useCache {
			clio.Debugw("refreshing credentials", "reason", "credential process cache is disabled via config")
		}

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

		credentials, err := profile.AssumeTerminal(c.Context, cfaws.ConfigOpts{Duration: duration, UsingCredentialProcess: true, CredentialProcessAutoLogin: autoLogin})
		var nae cfaws.NoAccessError

		// ensuring access is currently only supported for profiles using IAM Identity Center.
		if errors.As(err, &nae) && profile.AWSConfig.SSOAccountID != "" {
			clio.Debugw("received a NoAccessError", "wrapped_error", nae.Err)
			_, err = tryEnsureAccess(ctx, profile)
			if err != nil {
				return fmt.Errorf("error while ensuring access: %w: %w", err, nae)
			}
		}
		if err != nil {
			clio.Info("errrrerre")
			return errors.Wrap(err, "bagdaddadaad")
		}
		if !cfg.DisableCredentialProcessCache {
			clio.Debugw("storing refreshed credentials in credential process cache", "expires", credentials.Expires.String(), "canExpire", credentials.CanExpire, "timeNow", time.Now().String())
			if err := secureSessionCredentialStorage.StoreCredentials(profileName, credentials); err != nil {
				return err
			}
		}

		return printCredentials(credentials)
	},
}

func printCredentials(creds aws.Credentials) error {
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
}

func tryEnsureAccess(ctx context.Context, profile *cfaws.Profile) (bool, error) {
	if profile.AWSConfig.SSOAccountID == "" {
		clio.Debugw("skipping ensuring access", "reason", "SSOAccountID was empty")
		return false, nil
	}

	if profile.AWSConfig.SSORoleName == "" {
		clio.Debugw("skipping ensuring access", "reason", "SSORoleName was empty")
		return false, nil
	}

	// if Common Fate is configured, try and ensure access.
	cfg, err := sdkconfig.LoadDefault(ctx)
	if err != nil {
		return false, err
	}

	accessclient := access.NewFromConfig(cfg)

	_, err = accessclient.BatchEnsure(ctx, connect.NewRequest(&accessv1alpha1.BatchEnsureRequest{
		Entitlements: []*accessv1alpha1.EntitlementInput{
			{
				Target: &accessv1alpha1.Specifier{
					Specify: &accessv1alpha1.Specifier_Eid{
						Eid: &entityv1alpha1.EID{
							Type: "AWS::Account",
							Id:   profile.AWSConfig.SSOAccountID,
						},
					},
				},
				Role: &accessv1alpha1.Specifier{
					Specify: &accessv1alpha1.Specifier_Lookup{
						Lookup: profile.AWSConfig.SSORoleName,
					},
				},
			},
		},
	}))
	if err != nil {
		return false, err
	}

	return true, nil
}

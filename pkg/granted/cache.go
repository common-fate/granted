package granted

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/AlecAivazis/survey/v2"
	"github.com/common-fate/clio"
	"github.com/common-fate/granted/pkg/securestorage"
	"github.com/common-fate/granted/pkg/testable"
	"github.com/urfave/cli/v2"
)

var CacheCommand = cli.Command{
	Name:        "cache",
	Usage:       "Manage your cached credentials that are stored in secure storage",
	Subcommands: []*cli.Command{&ClearCommand, &ListCommand},
}

var ListCommand = cli.Command{
	Name:  "list",
	Usage: "List currently cached credentials and secure storage type",
	Action: func(c *cli.Context) error {
		storageToNameMap := map[string]securestorage.SecureStorage{
			"aws-iam-credentials": securestorage.NewSecureIAMCredentialStorage().SecureStorage,
			"sso-token":           securestorage.NewSecureSSOTokenStorage().SecureStorage,
			"session-credentials": securestorage.NewSecureSessionCredentialStorage().SecureStorage,
		}

		tw := tabwriter.NewWriter(os.Stderr, 10, 1, 5, ' ', 0)
		headers := strings.Join([]string{"STORAGE TYPE", "KEY"}, "\t")
		fmt.Fprintln(tw, headers)

		for storageName, v := range storageToNameMap {

			keys, err := v.ListKeys()
			if err != nil {
				return err
			}

			for _, key := range keys {
				tabbed := strings.Join([]string{storageName, key}, "\t")
				fmt.Fprintln(tw, tabbed)
			}

		}

		tw.Flush()

		return nil
	},
}

var ClearCommand = cli.Command{
	Name:  "clear",
	Usage: "Clear cached credential from the secure storage",
	Flags: []cli.Flag{
		&cli.BoolFlag{Name: "all", Usage: "clears all of the cached credentials from storage"},
		&cli.StringFlag{Name: "storage", Usage: "Specify the storage type"},
		&cli.StringFlag{Name: "profile", Usage: "Specify the profile name of the credential which should be cleared"},
	},
	Action: func(c *cli.Context) error {

		withStdio := survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)

		selection := c.String("storage")
		if selection == "" {
			in := survey.Select{
				Message: "Select which secure storage would you like to clear cache from",
				Options: []string{"aws-iam-credentials", "sso-token", "session-credentials"},
			}
			clio.NewLine()
			err := testable.AskOne(&in, &selection, withStdio)
			if err != nil {
				return err
			}
		}

		clearAll := c.Bool("all")

		if clearAll {
			clio.Debugw("clear flag provided clearing cache for all credentials in storage", "storage", selection)
		}

		storageToNameMap := map[string]securestorage.SecureStorage{
			"aws-iam-credentials": securestorage.NewSecureIAMCredentialStorage().SecureStorage,
			"sso-token":           securestorage.NewSecureSSOTokenStorage().SecureStorage,
			"session-credentials": securestorage.NewSecureSessionCredentialStorage().SecureStorage,
		}

		// store the credentials in secure storage
		selectedStorage := storageToNameMap[selection]

		keys, err := selectedStorage.ListKeys()
		if err != nil {
			return err
		}

		if len(keys) == 0 {
			clio.Warnf("You do not have any cached credentials for %s storage", selection)
			return nil
		}

		if clearAll {
			for _, key := range keys {
				err = selectedStorage.Clear(key)
				if err != nil {
					return err
				}
			}
			clio.Infow("cleared cache for all credentials in storage", "storage", selection)
			return nil

		}

		selectedProfile := c.String("profile")
		if selectedProfile == "" {
			prompt := &survey.Select{
				Message: "Select the profile name you want to clear cache for",
				Options: keys,
			}
			err = survey.AskOne(prompt, &selectedProfile)
			if err != nil {
				return err
			}
		}

		err = selectedStorage.Clear(selectedProfile)
		if err != nil {
			return err
		}

		clio.Successf("successfully cleared the cached credentials for '%s'", selectedProfile)

		return nil
	},
}

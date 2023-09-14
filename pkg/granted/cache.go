package granted

import (
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/common-fate/clio"
	"github.com/common-fate/granted/pkg/securestorage"
	"github.com/common-fate/granted/pkg/testable"
	"github.com/urfave/cli/v2"
)

var CacheCommand = cli.Command{
	Name:        "cache",
	Usage:       "Manage your cached credentials in secure storage",
	Subcommands: []*cli.Command{&ClearCommand},
}

var ClearCommand = cli.Command{
	Name: "clear",
	Action: func(c *cli.Context) error {
		withStdio := survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)
		in := survey.Select{
			Message: "Select which secure storage would you like to clear cache from",
			Options: []string{"aws-iam-credentials", "sso-token", "session-credentials"},
		}
		var selection string
		clio.NewLine()
		err := testable.AskOne(&in, &selection, withStdio)
		if err != nil {
			return err
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

		prompt := &survey.Select{
			Message: "Select the profile name you want to clear cache for",
			Options: keys,
		}
		var selectedProfile string
		err = survey.AskOne(prompt, &selectedProfile)
		if err != nil {
			return err
		}

		err = selectedStorage.Clear(selectedProfile)
		if err != nil {
			return err
		}

		clio.Successf("successfully cleared the cached credentials for '%s'", selectedProfile)

		return nil
	},
}

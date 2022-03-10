package granted

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/common-fate/granted/pkg/cfaws"
	"github.com/common-fate/granted/pkg/credstore"
	"github.com/common-fate/granted/pkg/testable"
	"github.com/urfave/cli/v2"
)

var TokenCommand = cli.Command{
	Name:        "token",
	Usage:       "Manage aws access tokens",
	Subcommands: []*cli.Command{&TokenListCommand, &ClearTokensCommand},
	Action:      TokenListCommand.Action,
}

var TokenListCommand = cli.Command{
	Name:  "list",
	Usage: "Lists all access tokens saved in the keyring",
	Action: func(ctx *cli.Context) error {
		tokens, err := credstore.List()
		if err != nil {
			return err
		}
		var max int
		for _, token := range tokens {
			if len(token.Key) > max {
				max = len(token.Key)
			}
		}

		for _, token := range tokens {
			fmt.Fprintf(os.Stderr, "%s\n", fmt.Sprintf("%-*s (%s)", max, token.Key, token.Description))
		}
		return nil
	},
}

// granted token -> lists all tick
// granted token list -> lists all tick
// granted token clear -> prompts for selection // promt confirm?
// granted token clear --all or -a -> clear all
// granted token clear profilename -> clear profile
// granted token clear profilename --confirm -> skip confirm prompt

func MapTokens(ctx context.Context) (map[string][]string, error) {
	keys, err := credstore.ListKeys()
	if err != nil {
		return nil, err
	}

	conf, err := cfaws.GetProfilesFromDefaultSharedConfig(ctx)
	if err != nil {
		return nil, err
	}
	startUrlMap := make(map[string][]string)
	for _, k := range keys {
		startUrlMap[k] = []string{}
	}
	a := &cfaws.AwsSsoAssumer{}
	for _, c := range conf {
		if c.ProfileType == a.Type() {
			ssoUrl := c.AWSConfig.SSOStartURL
			if len(c.Parents) != 0 {
				ssoUrl = c.Parents[0].AWSConfig.SSOStartURL
			}
			// Don't add any profiles which are not in the keyring already
			if _, ok := startUrlMap[ssoUrl]; ok {
				startUrlMap[ssoUrl] = append(startUrlMap[ssoUrl], c.Name)
			}
		}
	}
	return startUrlMap, nil
}

var ClearTokensCommand = cli.Command{
	Name:  "clear",
	Usage: "Remove a selected token from the keyring",
	Flags: []cli.Flag{
		&cli.BoolFlag{Name: "all", Aliases: []string{"a"}, Usage: "Remove all saved tokens from keyring"},
	},
	Action: func(c *cli.Context) error {

		if c.Bool("all") {
			err := credstore.ClearAll()
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Cleared all saved tokens")
			return nil
		}
		var selection string

		if c.Args().First() != "" {
			selection = c.Args().First()
		}

		startUrlMap, err := MapTokens(c.Context)
		if err != nil {
			return err
		}
		if selection == "" {
			var max int
			for k := range startUrlMap {
				if len(k) > max {
					max = len(k)
				}
			}
			selectionsMap := make(map[string]string)
			tokenList := []string{}
			for k, profiles := range startUrlMap {
				stringKey := fmt.Sprintf("%-*s (%s)", max, k, strings.Join(profiles, ", "))
				tokenList = append(tokenList, stringKey)
				selectionsMap[stringKey] = k
			}
			withStdio := survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)
			in := survey.Select{
				Message: "Select a token to remove from keyring",
				Options: tokenList,
			}
			fmt.Fprintln(os.Stderr)
			var out string
			err = testable.AskOne(&in, &out, withStdio)
			if err != nil {
				return err
			}
			selection = selectionsMap[out]
		}
		err = credstore.Clear(selection)
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Cleared %s", selection)
		return nil
	},
}

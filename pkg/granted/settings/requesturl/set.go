package requesturl

import (
	"fmt"
	"net/url"
	"os"

	"github.com/AlecAivazis/survey/v2"
	grantedConfig "github.com/common-fate/granted/pkg/config"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

var setRequestURLCommand = cli.Command{
	Name:  "set",
	Usage: "Set the request URL for Granted Approvals",
	Action: func(c *cli.Context) error {
		var approvalsURL string
		gConf, err := grantedConfig.Load()
		if err != nil {
			return errors.Wrap(err, "unable to load granted config")
		}

		approvalsURL = c.Args().First()
		if approvalsURL == "" {
			in := &survey.Input{
				Message: "What is the URL of your Granted Approvals deployment?",
				Help:    "URL for your Granted Approvals webapp from where users can request access \n for e.g: https://example.com",
			}
			withStdio := survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)
			err := survey.AskOne(in, &approvalsURL, withStdio)
			if err != nil {
				return err
			}

			if approvalsURL == "" {
				fmt.Println("Granted Approval URL not provided. Command aborted.")
				return nil
			}
		}

		parsedURL, err := url.ParseRequestURI(approvalsURL)
		if err != nil {
			return errors.Wrap(err, "unable to parse provided URL with err")
		}

		gConf.AccessRequestURL = parsedURL.String()
		err = gConf.Save()
		if err != nil {
			return err
		}

		fmt.Printf("Request url for Granted Approvals has been set to '%s'\n", approvalsURL)
		return nil
	},
}

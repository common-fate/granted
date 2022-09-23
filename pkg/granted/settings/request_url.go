package settings

import (
	"fmt"
	"os"

	"net/url"

	"github.com/AlecAivazis/survey/v2"
	grantedConfig "github.com/common-fate/granted/pkg/config"
	"github.com/pkg/errors"

	"github.com/urfave/cli/v2"
)

var RequestURLCommand = cli.Command{
	Name:  "request-url",
	Usage: "Set the request url for credential_process command (connection to Grante dApprovals)",
	Flags: []cli.Flag{
		&cli.BoolFlag{Name: "clear", Usage: "Clears the current request url"},
		&cli.StringFlag{Name: "set", Usage: "Bypass the interactive prompt and set the request url"},
	},
	Action: func(c *cli.Context) error {
		var approvalsURL string
		gConf, err := grantedConfig.Load()
		if err != nil {
			return errors.Wrap(err, "loading Granted config")
		}

		if c.Bool("clear") {
			gConf.AccessRequestURL = ""
			if err := gConf.Save(); err != nil {
				return errors.Wrap(err, "saving config")
			}
			fmt.Println("Successfully cleared the request url")
			return nil
		}

		if gConf.AccessRequestURL == "" && c.String("set") == "" {
			in := &survey.Input{
				Message: "What is the URL of your Granted Approvals deployment?",
				Help:    "i.e. https://commonfate.approvals.dev",
			}
			withStdio := survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)
			err := survey.AskOne(in, &approvalsURL, withStdio)
			if err != nil {
				return err
			}
			if approvalsURL == "" {
				return errors.New("cancelled setup process")
			}
			_, err = url.Parse(approvalsURL)
			if err != nil {
				return errors.Wrap(err, "parsing URL")
			}

			gConf.AccessRequestURL = approvalsURL
			err = gConf.Save()
			if err != nil {
				return err
			}
			fmt.Println("Successfully set the request url")
		} else if c.String("set") != "" {
			approvalsURL = c.String("set")
			_, err = url.Parse(approvalsURL)
			if err != nil {
				return errors.Wrap(err, "parsing URL")
			}
			gConf.AccessRequestURL = approvalsURL
			err = gConf.Save()
			if err != nil {
				return err
			}
			fmt.Println("Successfully set the request url")
		} else if gConf.AccessRequestURL != "" {
			fmt.Println("The current request url is:", gConf.AccessRequestURL)
		}
		return nil
	},
}

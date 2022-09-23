package settings

import (
	"errors"
	"fmt"
	"os"

	urlLib "net/url"

	"github.com/AlecAivazis/survey/v2"
	grantedConfig "github.com/common-fate/granted/pkg/config"

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

		var url string
		gConf, err := grantedConfig.Load()
		if err != nil {
			return errors.New("failed to load Config for GrantedApprovalsURL")
		}

		if c.Bool("clear") {
			gConf.GrantedApprovalsURL = ""
			if err := gConf.Save(); err != nil {
				return errors.New("failed to save Config for GrantedApprovalsURL")
			}
			fmt.Println("Successfully cleared the request url")
			return nil
		}

		if gConf.GrantedApprovalsURL == "" && c.String("set") == "" {
			in := &survey.Input{
				Message: "What is the base url of your Granted Approvals deployment\n",
				Help:    "i.e. https://commonfate.approvals.dev",
			}
			withStdio := survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)
			err := survey.AskOne(in, &url, withStdio)
			if err != nil {
				return err
			}
			if url == "" {
				return errors.New("cancelled setup process")
			}
			_, err = urlLib.Parse(url)
			if err != nil {
				return errors.New("invalid url")
			}

			gConf.GrantedApprovalsURL = url
			err = gConf.Save()
			if err != nil {
				return err
			}
			fmt.Println("Successfully set the request url")
		} else if c.String("set") != "" {
			url = c.String("set")
			_, err = urlLib.Parse(url)
			if err != nil {
				return errors.New("invalid url")
			}
			gConf.GrantedApprovalsURL = url
			err = gConf.Save()
			if err != nil {
				return err
			}
			fmt.Println("Successfully set the request url")
		} else if gConf.GrantedApprovalsURL != "" {
			fmt.Println("The current request url is:", gConf.GrantedApprovalsURL)
		}
		return nil
	},
}

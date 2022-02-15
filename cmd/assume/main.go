package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"

	"github.com/AlecAivazis/survey/v2"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/bigkevmcd/go-configparser"
	"github.com/common-fate/granted/pkg/alias"
	"github.com/pkg/browser"

	"github.com/common-fate/granted/internal/build"
	"github.com/common-fate/sig"
	"github.com/urfave/cli/v2"
)

func main() {
	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Fprintf(os.Stderr, "Granted v%s\n", build.Version)
	}

	flags := []cli.Flag{
		&cli.BoolFlag{Name: "console", Aliases: []string{"c"}, Usage: "Open a web console to the role"},
		&cli.BoolFlag{Name: "extension", Aliases: []string{"e"}, Usage: "Open a web console to the role using the Granted Containers extension"},
		&cli.BoolFlag{Name: "verbose", Usage: "Log debug messages"},
	}

	app := &cli.App{
		Name:        "assume",
		Usage:       "https://granted.dev",
		UsageText:   "assume [role] [account]",
		Version:     build.Version,
		HideVersion: false,
		Flags:       flags,
		Action:      AssumeCommand,
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func AssumeCommand(c *cli.Context) error {

	// @TODO: ensure this works as desired, currently we're disabling with an env flag
	if os.Getenv("FORCE_NO_ALIAS") != "true" {
		err := alias.MustBeConfigured()
		if err != nil {
			return err
		}
	}

	// role := c.Args().Get(0)
	// accountInput := c.Args().Get(1)

	// fetch the parsed config file
	configPath := config.DefaultSharedConfigFilename()
	config, err := configparser.NewConfigParserFromFile(configPath)
	if err != nil {
		return err
	}

	var profileMap = make(map[string]string)
	allProfileOptions := []string{}
	withStdio := survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)
	// .aws/config files are structured as follows,
	// We want to strip the profile_name i.e. [profile <profile_name>],
	//
	// [profile cf-dev]
	// sso_region=ap-southeast-2
	// ...
	// [profile cf-prod]
	// sso_region=ap-southeast-2
	// ...

	// Itterate through the config sections
	for _, section := range config.Sections() {

		// Check if the section is prefixed with 'profile '
		if section[0:7] == "profile" {

			// Strip 'profile ' from the section name
			awsProfile := section[8:]
			ssoID, err := config.Get(section, "sso_account_id")
			if err != nil {
				return err
			}

			value := fmt.Sprintf("%-16s%s:%s", awsProfile, "aws", ssoID)

			profileMap[awsProfile] = value
			allProfileOptions = append(allProfileOptions, value)
		}
	}

	// Replicate the logic from original assume fn.
	in := survey.Select{
		Options: allProfileOptions,
	}
	var profile string
	// TODO: see if we can use 'testable' here? Unhandled panic was being thrown
	err = survey.AskOne(&in, &profile, withStdio)
	if err != nil {
		return err
	}
	if profile != "" {
		// @NOTE: this is just ground work for the parent tickets
		// Currently we're not using the input, it's just being captured and logged
		fmt.Fprintf(os.Stderr, "ℹ️  Assume role with %s\n", profile)
	}

	var resbody sig.AssumeRoleResults
	// err = json.NewDecoder(ahres.Body).Decode(&resbody)
	// if err != nil {
	// 	return err
	// }

	if c.Bool("console") || c.Bool("extension") {
		sess := struct {
			SessionID    string `json:"sessionId"`
			SesssionKey  string `json:"sessionKey"`
			SessionToken string `json:"sessionToken"`
		}{
			SessionID:    resbody.AccessKeyID,
			SesssionKey:  resbody.SecretAccessKey,
			SessionToken: resbody.SessionToken,
		}
		sessJSON, err := json.Marshal(sess)
		if err != nil {
			return err
		}

		u := url.URL{
			Scheme: "https",
			Host:   "signin.aws.amazon.com",
			Path:   "/federation",
		}
		q := u.Query()
		q.Add("Action", "getSigninToken")
		q.Add("Session", string(sessJSON))
		u.RawQuery = q.Encode()

		res, err := http.Get(u.String())
		if err != nil {
			return err
		}
		if res.StatusCode != http.StatusOK {
			return fmt.Errorf("opening console failed with code %v", res.StatusCode)
		}

		token := struct {
			SigninToken string `json:"SigninToken"`
		}{}

		err = json.NewDecoder(res.Body).Decode(&token)
		if err != nil {
			return err
		}

		u = url.URL{
			Scheme: "https",
			Host:   "signin.aws.amazon.com",
			Path:   "/federation",
		}
		q = u.Query()
		q.Add("Action", "login")
		q.Add("Issuer", "")
		q.Add("Destination", "https://console.aws.amazon.com/console/home")
		q.Add("SigninToken", token.SigninToken)
		u.RawQuery = q.Encode()

		var fullyQualifiedAccount string

		if c.Bool("extension") {
			tabURL := fmt.Sprintf("ext+granted-containers:name=%s:%s (ap-southeast-2)&url=%s", "@TODO: fix to use SSO roles", fullyQualifiedAccount, url.QueryEscape(u.String()))
			cmd := exec.Command("/Applications/Firefox.app/Contents/MacOS/firefox",
				"--new-tab",
				tabURL)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			return cmd.Run()
		} else {
			return browser.OpenURL(u.String())
		}
	}
	// else {
	// fmt.Printf("GrantedAssume %s %s %s", resbody.AccessKeyID, resbody.SecretAccessKey, resbody.SessionToken)
	// fmt.Fprintf(os.Stderr, "\033[32m[%s] session credentials will expire %s\033[0m\n", role, resbody.Expiration.Local().String())
	// }

	return nil
}

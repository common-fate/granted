package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"

	"github.com/AlecAivazis/survey/v2"
	teamv1alpha1 "github.com/common-fate/cf-protos/gen/proto/go/team/v1alpha1"
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

	role := c.Args().Get(0)
	accountInput := c.Args().Get(1)

	withStdio := survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)
	allRoleOptions := []string{}

	type RoleOptionValue struct {
		roleId     *teamv1alpha1.Role
		providerId string
		customerId string
	}

	roleOptionMap := make(map[string]RoleOptionValue)
	// roleMap := make(map[string]*teamv1alpha1.Role)

	// for _, role := range rolesRes.Roles {
	// 	// PrintF formatting to align to columns
	// 	for _, account := range role.Account {
	// 		stringKey := fmt.Sprintf("%-12s%s:%s", role.Id, account.Provider, account.AccountId)
	// 		allRoleOptions = append(allRoleOptions, stringKey)
	// 		// append to roleOptionMap
	// 		roleOptionMap[stringKey] = RoleOptionValue{
	// 			roleId:     role,
	// 			providerId: account.Provider,
	// 			customerId: account.AccountId,
	// 		}
	// 	}
	// 	roleMap[role.Id] = role

	// }

	if role == "" && accountInput == "" {
		in := survey.Select{
			Options: allRoleOptions,
		}
		var roleacc string
		err := survey.AskOne(&in, &roleacc, withStdio)
		if err != nil {
			return err
		}
		accountInput = roleOptionMap[roleacc].customerId
		role = roleOptionMap[roleacc].roleId.Id
	}

	// matchedRole, ok := roleMap[role]
	// if !ok {
	// 	return errors.New("this role either doesn't exist, or you don't have access to it")
	// }

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
			tabURL := fmt.Sprintf("ext+granted-containers:name=%s:%s (ap-southeast-2)&url=%s", role, fullyQualifiedAccount, url.QueryEscape(u.String()))
			cmd := exec.Command("/Applications/Firefox.app/Contents/MacOS/firefox",
				"--new-tab",
				tabURL)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			return cmd.Run()
		} else {
			return browser.OpenURL(u.String())
		}
	} else {
		fmt.Printf("GrantedAssume %s %s %s", resbody.AccessKeyID, resbody.SecretAccessKey, resbody.SessionToken)
		fmt.Fprintf(os.Stderr, "\033[32m[%s] session credentials will expire %s\033[0m\n", role, resbody.Expiration.Local().String())
	}

	return nil
}

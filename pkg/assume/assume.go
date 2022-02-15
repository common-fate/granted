package assume

import (
	"fmt"
	"os"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/common-fate/granted/pkg/cfaws"
	"github.com/common-fate/granted/pkg/testable"
	"github.com/urfave/cli/v2"
)

func AssumeCommand(c *cli.Context) error {

	withStdio := survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)
	awsProfiles, err := cfaws.GetProfilesFromDefaultSharedConfig(c.Context)
	if err != nil {
		return err
	}
	// Replicate the logic from original assume fn.
	in := survey.Select{
		Options: awsProfiles.ProfileNames(),
	}
	var profile string
	err = testable.AskOne(&in, &profile, withStdio)
	if err != nil {
		return err
	}
	if profile != "" {
		// @NOTE: this is just ground work for the parent tickets
		// Currently we're not using the input, it's just being captured and logged
		fmt.Fprintf(os.Stderr, "ℹ️  Assume role with %s\n", profile)
	}
	role := "rolename goes here"
	account := "123456789120"
	accessKeyID := "todo"
	secretAccessKey := "todo"
	sessionToken := "todo"
	expiration := time.Now().Add(time.Hour)

	sess := Session{SessionID: accessKeyID, SesssionKey: secretAccessKey, SessionToken: sessionToken}
	labels := RoleLabels{Role: role, Account: account}
	if c.Bool("console") {
		return LaunchConsoleSession(sess, labels, BrowserDefault)
	} else if c.Bool("extension") {
		return LaunchConsoleSession(sess, labels, BrowerFirefox)
	} else if c.Bool("chrome") {
		return LaunchConsoleSession(sess, labels, BrowserChrome)
	} else {
		fmt.Printf("GrantedAssume %s %s %s", accessKeyID, secretAccessKey, sessionToken)
		fmt.Fprintf(os.Stderr, "\033[32m[%s] session credentials will expire %s\033[0m\n", role, expiration.Local().String())
	}

	return nil
}

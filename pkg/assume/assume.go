package assume

import (
	"fmt"
	"os"

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
	var p string
	err = testable.AskOne(&in, &p, withStdio)
	if err != nil {
		return err
	}

	profile := awsProfiles[p]

	fmt.Fprintf(os.Stderr, "ℹ️  Assume role with %s\n", profile.Name)

	creds, err := profile.Assume(c.Context)
	if err != nil {
		return err
	}

	accessKeyID := creds.AccessKeyID
	secretAccessKey := creds.SecretAccessKey
	sessionToken := creds.SessionToken
	expiration := creds.Expires

	sess := Session{SessionID: accessKeyID, SesssionKey: secretAccessKey, SessionToken: sessionToken}

	// you cannot login to the console with iam??
	if profile.ProfileType != cfaws.ProfileTypeIAM {

		// these are just labels for the tabs so we may need to updates these for the sso role context
		role := "todo"
		account := "todo"
		labels := RoleLabels{Role: role, Account: account}
		if c.Bool("console") {
			return LaunchConsoleSession(sess, labels, BrowserDefault)
		} else if c.Bool("extension") {
			return LaunchConsoleSession(sess, labels, BrowerFirefox)
		} else if c.Bool("chrome") {
			return LaunchConsoleSession(sess, labels, BrowserChrome)
		}
	} else {
		// DO NOT MODIFY, this like interacts with the shell script that wraps the assume command, the shell script is what configures your shell environment vars
		fmt.Printf("GrantedAssume %s %s %s", creds.AccessKeyID, creds.SecretAccessKey, creds.SessionToken)
		fmt.Fprintf(os.Stderr, "\033[32m[%s] session credentials will expire %s\033[0m\n", profile.Name, expiration.Local().String())
	}

	return nil
}

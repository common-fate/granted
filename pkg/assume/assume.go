package assume

import (
	"fmt"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/common-fate/granted/pkg/cfaws"
	"github.com/common-fate/granted/pkg/testable"
	"github.com/fatih/color"
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

	// these are just labels for the tabs so we may need to updates these for the sso role context
	role := "todo"
	account := "todo"
	labels := RoleLabels{Role: role, Account: account}

	isIamWithoutAssumedRole := profile.ProfileType == cfaws.ProfileTypeIAM && profile.RawConfig.RoleARN == ""
	openBrower := c.Bool("console") || c.Bool("extension") || c.Bool("chrome")
	if openBrower && isIamWithoutAssumedRole {
		fmt.Fprintf(os.Stderr, "Cannot open a browser session for profile: %s because it does not assume a role", profile.Name)
	} else if openBrower {
		if c.Bool("extension") {
			return LaunchConsoleSession(sess, labels, BrowerFirefox)
		} else if c.Bool("chrome") {
			return LaunchConsoleSession(sess, labels, BrowserChrome)
		} else {
			return LaunchConsoleSession(sess, labels, BrowserDefault)
		}
	} else {
		// DO NOT MODIFY, this like interacts with the shell script that wraps the assume command, the shell script is what configures your shell environment vars
		fmt.Printf("GrantedAssume %s %s %s", creds.AccessKeyID, creds.SecretAccessKey, creds.SessionToken)
		green := color.New(color.FgGreen)
		if creds.CanExpire {
			green.Fprintf(os.Stderr, "\n[%s] session credentials will expire %s\n", profile.Name, expiration.Local().String())
		} else {
			green.Fprintf(os.Stderr, "\n[%s] session credentials ready\n", profile.Name)
		}
	}

	return nil
}

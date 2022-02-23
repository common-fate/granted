package assume

import (
	"fmt"
	"sync"

	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/common-fate/granted/pkg/browsers"
	"github.com/common-fate/granted/pkg/cfaws"
	"github.com/common-fate/granted/pkg/testable"
	"github.com/urfave/cli/v2"
)

func AssumeCommand(c *cli.Context) error {
	var wg sync.WaitGroup

	withStdio := survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)
	awsProfiles, err := cfaws.GetProfilesFromDefaultSharedConfig(c.Context)
	if err != nil {
		return err
	}

	var profile *cfaws.CFSharedConfig
	inProfile := c.Args().First()
	if inProfile != "" {
		var ok bool
		if profile, ok = awsProfiles[inProfile]; !ok {
			fmt.Fprintf(os.Stderr, "%s does not match any profiles in your AWS config\n", inProfile)
		} else {
			// background task to update the frecency cache
			wg.Add(1)
			go func() {
				cfaws.UpdateFrecencyCache(inProfile)
				wg.Done()
			}()
		}
	}
	//set the sesh creds using the active role if we have one and the flag is set
	if c.Bool("active-role") && os.Getenv("GRANTED_AWS_ROLE_PROFILE") != "" {
		//try opening using the active role
		fmt.Fprintf(os.Stderr, "Attempting to open using active role...\n")

		profileName := os.Getenv("GRANTED_AWS_ROLE_PROFILE")

		profile = awsProfiles[profileName]

	}

	if profile == nil && !c.Bool("active-role") {

		fr, profiles := awsProfiles.GetFrecentProfiles()
		fmt.Fprintln(os.Stderr, "")
		// Replicate the logic from original assume fn.
		in := survey.Select{
			Message: "Please select the profile you would like to assume:",
			Options: profiles,
		}
		var p string
		err = testable.AskOne(&in, &p, withStdio)
		if err != nil {
			return err
		}

		profile = awsProfiles[p]
		// background task to update the frecency cache
		wg.Add(1)
		go func() {
			fr.Update(p)
			wg.Done()
		}()
	}
	// ensure that frecency has finished updating before returning from this function
	defer wg.Wait()

	creds, err := profile.Assume(c.Context)
	if err != nil {
		return err
	}

	accessKeyID := creds.AccessKeyID
	secretAccessKey := creds.SecretAccessKey
	sessionToken := creds.SessionToken
	expiration := creds.Expires

	sess := browsers.Session{SessionID: accessKeyID, SesssionKey: secretAccessKey, SessionToken: sessionToken}

	// these are just labels for the tabs so we may need to updates these for the sso role context
	labels := browsers.RoleLabels{Profile: profile.Name}

	isIamWithoutAssumedRole := profile.ProfileType == cfaws.ProfileTypeIAM && profile.RawConfig.RoleARN == ""
	openBrower := c.Bool("console") || c.Bool("active-role")
	if openBrower && isIamWithoutAssumedRole {
		// @TODO check if we can launch the console as an IAM user
		fmt.Fprintf(os.Stderr, "\nCannot open a browser session for profile: %s because it does not assume a role\n", profile.Name)
	} else if openBrower {
		service := c.String("service")
		region := c.String("region")
		
		labels.Region = region
    fmt.Fprintf(os.Stderr, "\nOpening a console for %s in your browser...\n", profile.Name)
		return browsers.LaunchConsoleSession(sess, labels, service, region)
	} else {
		region, _, err := profile.Region(c.Context)
		if err != nil {
			region = "None"
		}
		// DO NOT REMOVE, this interacts with the shell script that wraps the assume command, the shell script is what configures your shell environment vars
		// to export more environment variables, add then in the assume and assume.fish scripts then append them to this printf
		fmt.Printf("GrantedAssume %s %s %s %s %s", creds.AccessKeyID, creds.SecretAccessKey, creds.SessionToken, profile.Name, region)
		if creds.CanExpire {
			fmt.Fprintf(os.Stderr, "\033[32m\n[%s] session credentials will expire %s\033[0m\n", profile.Name, expiration.Local().String())
		} else {
			fmt.Fprintf(os.Stderr, "\033[32m\n[%s] session credentials ready\033[0m\n", profile.Name)
		}
	}

	return nil
}

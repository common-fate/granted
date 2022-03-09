package assume

import (
	"fmt"
	"sync"
	"time"

	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/common-fate/granted/pkg/browsers"
	"github.com/common-fate/granted/pkg/cfaws"
	"github.com/common-fate/granted/pkg/config"
	"github.com/common-fate/granted/pkg/debug"
	"github.com/common-fate/granted/pkg/testable"
	cfflags "github.com/common-fate/granted/pkg/urfav_overrides"
	"github.com/urfave/cli/v2"
)

func AssumeCommand(c *cli.Context) error {
	// this custom behavious allows flags to be passed on either side of the role arg
	// to access flags in this command, use assumeFlags.String("region") etc instead of c.String("region")
	assumeFlags, err := cfflags.New("assumeFlags", GlobalFlags(), c)
	if err != nil {
		return err
	}
	var wg sync.WaitGroup
	activeRoleProfile := assumeFlags.String("granted-active-aws-role-profile")
	activeRoleFlag := assumeFlags.Bool("active-role")

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
	if activeRoleFlag && activeRoleProfile != "" {
		//try opening using the active role
		fmt.Fprintf(os.Stderr, "Attempting to open using active role...\n")
		profile = awsProfiles[activeRoleProfile]
		if profile == nil {
			debug.Fprintf(debug.VerbosityDebug, os.Stderr, "failed to find a profile matching AWS_PROFILE=%s when using the active-profile flag", activeRoleProfile)
		}

	}

	// if profile is still nil here, then prompt to select a profile

	if profile == nil {

		fr, profiles := awsProfiles.GetFrecentProfiles()
		fmt.Fprintln(os.Stderr, "")
		// Replicate the logic from original assume fn.
		in := survey.Select{
			Message: "Please select the profile you would like to assume:",
			Options: profiles,
		}
		if len(profiles) == 0 {
			fmt.Fprintln(os.Stderr, "ℹ️ Granted couldn't find any aws roles")
			fmt.Fprintln(os.Stderr, "You can add roles to your aws config by following our guide: ")
			fmt.Fprintln(os.Stderr, "https://granted.dev/awsconfig")
			return nil
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

	region, _, err := profile.Region(c.Context)
	if err != nil {
		return err
	}

	if assumeFlags.Bool("url") {
		//dont want to open the browser just return the link
		labels := browsers.RoleLabels{Profile: profile.Name}

		var creds aws.Credentials

		creds, err = profile.AssumeConsole(c.Context, assumeFlags.StringSlice("pass-through"))
		if err != nil {
			return err
		}

		service := assumeFlags.String("service")
		if assumeFlags.String("region") != "" {
			region = assumeFlags.String("region")
		}

		labels.Region = region
		labels.Service = service
		url, err := browsers.MakeUrl(browsers.SessionFromCredentials(creds), labels, service, region)
		if err != nil {
			return err
		}

		cfg, _ := config.Load()

		if cfg.DefaultBrowser == browsers.FirefoxKey {
			url = browsers.MakeFirefoxContainerURL(url, labels)
			if err != nil {
				return err
			}
			fmt.Print(url)

		} else {
			fmt.Print(url)

		}

		return nil
	}

	//update role duration if the duration flag is set
	if assumeFlags.String("duration") != "" {
		dur, err := time.ParseDuration(assumeFlags.String("duration"))

		if err != nil {
			return err
		}
		profile.AWSConfig.RoleDurationSeconds = &dur
	}
	openBrower := assumeFlags.Bool("console") || assumeFlags.Bool("active-role")
	if openBrower {
		// these are just labels for the tabs so we may need to updates these for the sso role context
		labels := browsers.RoleLabels{Profile: profile.Name}

		var creds aws.Credentials

		creds, err = profile.AssumeConsole(c.Context, assumeFlags.StringSlice("pass-through"))
		if err != nil {
			return err
		}

		service := assumeFlags.String("service")
		if assumeFlags.String("region") != "" {
			region = assumeFlags.String("region")
		}

		labels.Region = region
		labels.Service = service
		browsers.PromoteUseFlags(labels)
		fmt.Fprintf(os.Stderr, "\nOpening a console for %s in your browser...\n", profile.Name)
		return browsers.LaunchConsoleSession(browsers.SessionFromCredentials(creds), labels, service, region)
	} else {
		creds, err := profile.AssumeTerminal(c.Context, assumeFlags.StringSlice("pass-through"))
		if err != nil {
			return err
		}
		// DO NOT REMOVE, this interacts with the shell script that wraps the assume command, the shell script is what configures your shell environment vars
		// to export more environment variables, add then in the assume and assume.fish scripts then append them to this output preparation function
		// the shell script treats "None" as an emprty string and will not set a value for that positional output
		output := PrepareStringsForShellScript([]string{creds.AccessKeyID, creds.SecretAccessKey, creds.SessionToken, profile.Name, region})
		fmt.Printf("GrantedAssume %s %s %s %s %s", output...)
		if creds.CanExpire {
			duration := profile.AWSConfig.RoleDurationSeconds.Hours()
			fmt.Fprintf(os.Stderr, "\033[32m\n[%s](%s) session credentials will expire %s (%fh)\033[0m\n", profile.Name, region, creds.Expires.Local().String(), duration)
		} else {
			fmt.Fprintf(os.Stderr, "\033[32m\n[%s](%s) session credentials ready\033[0m\n", profile.Name, region)
		}
	}

	return nil
}

// PrepareCredentialsForShellScript will set empty values to "None", this is required by the shell script to identify which variables to unset
// it is also required to ensure that the return values are correctly split, e.g if sessionToken is "" then profile name will be used to set the session token environment variable
func PrepareStringsForShellScript(in []string) []interface{} {
	out := []interface{}{}
	for _, s := range in {
		if s == "" {
			out = append(out, "None")
		} else {
			out = append(out, s)
		}

	}
	return out
}

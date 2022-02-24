package assume

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"strings"
	"sync"

	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/common-fate/granted/pkg/browsers"
	"github.com/common-fate/granted/pkg/cfaws"
	"github.com/common-fate/granted/pkg/testable"
	"github.com/urfave/cli/v2"
)

func flagSet(name string, flags []cli.Flag) (*flag.FlagSet, error) {
	set := flag.NewFlagSet(name, flag.ContinueOnError)

	for _, f := range flags {
		if err := f.Apply(set); err != nil {
			return nil, err
		}
	}
	set.SetOutput(ioutil.Discard)
	return set, nil
}
func copyFlag(name string, ff *flag.Flag, set *flag.FlagSet) {
	switch ff.Value.(type) {
	case cli.Serializer:
		_ = set.Set(name, ff.Value.(cli.Serializer).Serialize())
	default:
		_ = set.Set(name, ff.Value.String())
	}
}

func normalizeFlags(flags []cli.Flag, set *flag.FlagSet) error {
	visited := make(map[string]bool)
	set.Visit(func(f *flag.Flag) {
		visited[f.Name] = true
	})
	for _, f := range flags {
		parts := f.Names()
		if len(parts) == 1 {
			continue
		}
		var ff *flag.Flag
		for _, name := range parts {
			name = strings.Trim(name, " ")
			if visited[name] {
				if ff != nil {
					return errors.New("Cannot use two forms of the same flag: " + name + " " + ff.Name)
				}
				ff = set.Lookup(name)
			}
		}
		if ff == nil {
			continue
		}
		for _, name := range parts {
			name = strings.Trim(name, " ")
			if !visited[name] {
				copyFlag(name, ff, set)
			}
		}
	}
	return nil
}
func searchFS(name string, fs *flag.FlagSet) []string {
	for _, f := range GlobalFlags {
		for _, n := range f.Names() {
			if n == name {
				return f.Names()
			}
		}
	}
	return nil
}
func FSString(name string, set *flag.FlagSet) string {
	names := searchFS(name, set)
	for _, n := range names {
		f := set.Lookup(n)
		if f != nil {
			parsed, err := f.Value.String(), error(nil)
			if err != nil {
				return ""
			}
			if parsed != "" {
				return parsed
			}

		}
	}

	return ""
}

// func FSBool(name string, set *flag.FlagSet) bool {
// 	f := set.Lookup(name)
// 	if f != nil {
// 		parsed, err := strconv.ParseBool(f.Value.String()), error(nil)
// 		if err != nil {
// 			return false
// 		}
// 		return parsed
// 	}
// 	return false
// }

func AssumeCommand(c *cli.Context) error {
	// cfFlags := NewCfFlagset(c)
	// CFFlags.String("region")

	// af := &AssumeFlags{}
	fs, err := flagSet("flags", GlobalFlags)
	if err != nil {
		return err
	}
	o := os.Args
	ca := c.Args().Slice()
	_ = ca
	_ = o

	// context.Args() for this command will ONLY contain the role and any flags provided after the role
	// this slice of os.Args will only contain flags and not the role if it was provided
	ag := os.Args[1 : len(os.Args)-len(ca)]
	ag = append(ag, ca[1:]...)
	err = normalizeFlags(GlobalFlags, fs)
	if err != nil {
		return err
	}
	err = fs.Parse(ag)
	if err != nil {
		return err
	}
	region := FSString("region", fs)
	fmt.Printf("region: %v\n", region)
	// // register CLI flags for other components
	// // fs.StringVar(&af.region, "region", "", "the log level (must match go.uber.org/zap log levels)")
	// fs.StringVar(&af.region, "r", "", "the log level (must match go.uber.org/zap log levels)")
	// fs.BoolVar(&af.console, "c", false, "the log level (must match go.uber.org/zap log levels)")

	// err := fs.Parse(os.Args[4:])
	// if err != nil {
	// 	return err
	// }

	// var test *cli.Context
	// app := &cli.App{Flags: GlobalFlags, Action: func(c *cli.Context) error {
	// 	test = c
	// 	fmt.Fprintf(os.Stderr, "test.String(\"region\"): %v\n", test.String("region"))
	// 	return nil
	// }}
	// a := c.Args().Slice()
	// err := app.Run(a)
	// if err != nil {
	// 	return err
	// }
	// fmt.Fprintf(os.Stderr, "test.String(\"region\"): %v\n", test.String("region"))
	r := c.String("region")
	fmt.Printf("r: %v\n", r)
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
		fmt.Fprintf(os.Stderr, "Cannot open a browser session for profile: %s because it does not assume a role", profile.Name)
	} else if openBrower {
		service := c.String("service")
		region := c.String("region")

		labels.Region = region
		fmt.Fprintf(os.Stderr, "Opening a console for %s in your browser...", profile.Name)
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

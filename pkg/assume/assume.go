package assume

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/common-fate/granted/pkg/browsers"
	"github.com/common-fate/granted/pkg/cfaws"
	"github.com/common-fate/granted/pkg/config"
	"github.com/common-fate/granted/pkg/testable"
	cfflags "github.com/common-fate/granted/pkg/urfav_overrides"
	"github.com/fatih/color"
	"github.com/hako/durafmt"
	"github.com/urfave/cli/v2"
)

func AssumeCommand(c *cli.Context) error {
	// this custom behavious allows flags to be passed on either side of the role arg
	// to access flags in this command, use assumeFlags.String("region") etc instead of c.String("region")
	assumeFlags, err := cfflags.New("assumeFlags", GlobalFlags(), c)
	if err != nil {
		return err
	}

	if assumeFlags.String("exec") != "" && runtime.GOOS == "windows" {
		return fmt.Errorf("--exec flag is not currently supported on Windows. Let us know if you'd like support for this: https://github.com/common-fate/granted/issues/new")
	}
	activeRoleProfile := assumeFlags.String("active-aws-profile")
	activeRoleFlag := assumeFlags.Bool("active-role")
	var profile *cfaws.Profile
	if assumeFlags.Bool("sso") {
		profile, err = SSOProfileFromFlags(c)
		if err != nil {
			return err
		}
	} else if activeRoleFlag && os.Getenv("GRANTED_SSO") == "true" {
		profile, err = SSOProfileFromEnv()
		if err != nil {
			return err
		}
	} else {
		var wg sync.WaitGroup

		withStdio := survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)
		profiles, err := cfaws.LoadProfiles()
		if err != nil {
			return err
		}

		profileName := c.Args().First()
		if profileName != "" {
			if !profiles.HasProfile(profileName) {
				fmt.Fprintf(color.Error, "%s does not match any profiles in your AWS config or credentials files\n", profileName)
				profileName = ""
			}
		}

		//set the sesh creds using the active role if we have one and the flag is set
		if activeRoleFlag && activeRoleProfile != "" {
			if !profiles.HasProfile(activeRoleProfile) {
				fmt.Fprintf(color.Error, "you tried to use the -active-role flag but %s does not match any profiles in your AWS config or credentials files\n", activeRoleProfile)
			} else {
				profileName = activeRoleProfile
				fmt.Fprintf(color.Error, "using active profile: %s\n", profileName)
			}
		}
		if profileName != "" {
			// background task to update the frecency cache
			wg.Add(1)
			go func() {
				cfaws.UpdateFrecencyCache(profileName)
				wg.Done()
			}()
		}

		// if profile is still "" here, then prompt to select a profile
		if profileName == "" {
			//load config to check frecency enabled
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			fr, profileNames := profiles.GetFrecentProfiles()
			if cfg.Ordering == "Alphabetical" {
				profileNames = profiles.ProfileNames
			}
			fmt.Fprintln(color.Error, "")
			// Replicate the logic from original assume fn.
			in := survey.Select{
				Message: "Please select the profile you would like to assume:",
				Options: profileNames,
				Filter:  filterMultiToken,
			}
			if len(profileNames) == 0 {
				fmt.Fprintln(color.Error, "ℹ️ Granted couldn't find any aws roles")
				fmt.Fprintln(color.Error, "You can add roles to your aws config by following our guide: ")
				fmt.Fprintln(color.Error, "https://granted.dev/awsconfig")
				return nil
			}
			err = testable.AskOne(&in, &profileName, withStdio)
			if err != nil {
				return err
			}
			// background task to update the frecency cache
			wg.Add(1)
			go func() {
				fr.Update(profileName)
				wg.Done()
			}()
		}
		// ensure that frecency has finished updating before returning from this function
		defer wg.Wait()
		//finally, load the profile and initialise it, this builds the parent tree structure
		profile, err = profiles.LoadInitialisedProfile(c.Context, profileName)
		if err != nil {
			return err
		}
	}

	var region string
	// The region flag may be supplied in shorthand form, first check if the flag is set and expand the region
	// else use the profile region
	if assumeFlags.String("region") != "" {
		regionFlag := assumeFlags.String("region")
		region, err = cfaws.ExpandRegion(regionFlag)
		if err != nil {
			return fmt.Errorf("couldn't parse region %s: %v", region, err)
		}
	} else {
		region, err = profile.Region(c.Context)
		if err != nil {
			return err
		}
	}

	configOpts := cfaws.ConfigOpts{Duration: time.Hour}

	//attempt to get session duration from profile
	if profile.AWSConfig.RoleDurationSeconds != nil {
		configOpts.Duration = *profile.AWSConfig.RoleDurationSeconds
	}

	duration := assumeFlags.String("duration")
	if duration != "" {
		d, err := time.ParseDuration(duration)
		if err != nil {
			return err
		}
		configOpts.Duration = d
	}

	if len(assumeFlags.StringSlice("pass-through")) > 0 {
		configOpts.Args = assumeFlags.StringSlice("pass-through")
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	openBrower := !assumeFlags.Bool("env") && (assumeFlags.Bool("console") || assumeFlags.Bool("active-role") || assumeFlags.String("service") != "" || assumeFlags.Bool("url"))

	if openBrower {
		// these are just labels for the tabs so we may need to updates these for the sso role context

		browserOpts := browsers.BrowserOpts{Profile: profile.Name}
		service := assumeFlags.String("service")

		browserOpts.Region = region
		browserOpts.Service = service

		var creds aws.Credentials

		creds, err = profile.AssumeConsole(c.Context, configOpts)
		if err != nil {
			return err
		}

		if assumeFlags.Bool("url") || cfg.DefaultBrowser == browsers.StdoutKey || cfg.DefaultBrowser == browsers.FirefoxStdoutKey {
			url, err := browsers.MakeUrl(browsers.SessionFromCredentials(creds), browserOpts, service, region)
			if err != nil {
				return err
			}
			if cfg.DefaultBrowser == browsers.FirefoxKey || cfg.DefaultBrowser == browsers.FirefoxStdoutKey {
				url = browsers.MakeFirefoxContainerURL(url, browserOpts)
				if err != nil {
					return err
				}
			}
			// return the url via stdout through the cli wrapper script
			fmt.Print(MakeGrantedOutput(url))
		} else {
			browsers.PromoteUseFlags(browserOpts)
			fmt.Fprintf(color.Error, "\nOpening a console for %s in your browser...\n", profile.Name)
			return browsers.LaunchConsoleSession(browsers.SessionFromCredentials(creds), browserOpts, service, region)
		}

	} else {
		creds, err := profile.AssumeTerminal(c.Context, configOpts)
		if err != nil {
			return err
		}
		sessionExpiration := ""
		green := color.New(color.FgGreen)
		if creds.CanExpire {
			sessionExpiration = creds.Expires.Local().Format(time.RFC3339)
			// We add 10 seconds here as a fudge factor, the credentials will be a
			// few seconds old already.
			durationDescription := durafmt.Parse(time.Until(creds.Expires) + 10*time.Second).LimitFirstN(1).String()
			if os.Getenv("GRANTED_QUIET") != "true" {
				green.Fprintf(color.Error, "\n[%s](%s) session credentials will expire in %s\n", profile.Name, region, durationDescription)
			}
		} else if os.Getenv("GRANTED_QUIET") != "true" {
			green.Fprintf(color.Error, "\n[%s](%s) session credentials ready\n", profile.Name, region)
		}
		if assumeFlags.Bool("env") {
			err = cfaws.WriteCredentialsToDotenv(region, creds)
			if err != nil {
				return err
			}
			green.Fprintln(color.Error, "Exported credentials to .env file successfully")
		}

		if assumeFlags.Bool("export") {
			err = cfaws.ExportCredsToProfile(profile.Name, creds)
			if err != nil {
				return err
			}
			var profileName string
			if cfg.ExportCredentialSuffix != "" {
				profileName = profile.Name + "-" + cfg.ExportCredentialSuffix

			} else {
				profileName = profile.Name
				yellow := color.New(color.FgYellow)

				yellow.Fprintln(color.Error, "No credential suffix found. This can cause issues with using exported credentials if conflicting profiles exist. Run `granted settings export-suffix set` to set one.")

			}

			green.Fprintln(color.Error, fmt.Sprintf("Exported credentials to ~.aws/credentials file as %s successfully", profileName))
		}
		if assumeFlags.String("exec") != "" {
			return RunExecCommandWithCreds(assumeFlags.String("exec"), creds, region)
		}
		// DO NOT REMOVE, this interacts with the shell script that wraps the assume command, the shell script is what configures your shell environment vars
		// to export more environment variables, add then in the assume and assume.fish scripts then append them to this output preparation function
		// the shell script treats "None" as an emprty string and will not set a value for that positional output
		if assumeFlags.Bool("sso") {
			output := PrepareStringsForShellScript([]string{creds.AccessKeyID, creds.SecretAccessKey, creds.SessionToken, "", region, sessionExpiration, "true", profile.AWSConfig.SSOStartURL, profile.AWSConfig.SSORoleName, profile.AWSConfig.SSORegion, profile.AWSConfig.SSOAccountID})
			fmt.Printf("GrantedAssume %s %s %s %s %s %s %s %s %s %s %s", output...)
		} else {
			output := PrepareStringsForShellScript([]string{creds.AccessKeyID, creds.SecretAccessKey, creds.SessionToken, profile.Name, region, sessionExpiration, "false", "", "", "", ""})
			fmt.Printf("GrantedAssume %s %s %s %s %s %s %s %s %s %s %s", output...)
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

// RunExecCommandWithCreds takes in a command, which may be a program and arguments sperated by spaces
// it splits these then runs the command with teh credentials as the environment.
// The output of this is returned via the assume script to stdout so it may be processed further by piping
func RunExecCommandWithCreds(cmd string, creds aws.Credentials, region string) error {
	fmt.Print(MakeGrantedOutput(""))
	args := strings.Split(cmd, " ")
	c := exec.Command(args[0], args[1:]...)
	c.Stdout = os.Stdout
	c.Stderr = color.Error
	c.Env = append(os.Environ(), EnvKeys(creds, region)...)
	return c.Run()
}

// EnvKeys is used to set the env for the "exec" flag
func EnvKeys(creds aws.Credentials, region string) []string {
	return []string{"AWS_ACCESS_KEY_ID=" + creds.AccessKeyID,
		"AWS_SECRET_ACCESS_KEY=" + creds.SecretAccessKey,
		"AWS_SESSION_TOKEN=" + creds.SessionToken,
		"AWS_REGION=" + region}
}

// MakeGrantedOutput formats a string to match the requirements of granted output in the shell script
// Currently in windows, the grantedoutput is handled differently, as linux and mac support the exec cli flag whereas windows does not yet have support
// this method may be changed in future if we implement support for "--exec" in windows
func MakeGrantedOutput(s string) string {
	// if the GRANTED_ALIAS_CONFIGURED env variable isn't set,
	// we aren't running in the context of the `assume` shell script.
	// If this is the case, don't add a prefix to the output as we don't have the
	// wrapper shell script to parse it.
	if os.Getenv("GRANTED_ALIAS_CONFIGURED") != "true" {
		return ""
	}
	out := "GrantedOutput"
	if runtime.GOOS != "windows" {
		out += "\n"
	} else {
		out += " "
	}
	return out + s
}

func filterMultiToken(filterValue string, optValue string, optIndex int) bool {
	optValue = strings.ToLower(optValue)
	filters := strings.Split(strings.ToLower(filterValue), " ")
	for _, filter := range filters {
		if !strings.Contains(optValue, filter) {
			return false
		}
	}
	return true
}

package assume

import (
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"sync"
	"time"

	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/common-fate/granted/pkg/assumeprint"
	"github.com/common-fate/granted/pkg/browser"
	"github.com/common-fate/granted/pkg/cfaws"
	"github.com/common-fate/granted/pkg/config"
	"github.com/common-fate/granted/pkg/console"
	"github.com/common-fate/granted/pkg/forkprocess"
	"github.com/common-fate/granted/pkg/launcher"
	"github.com/common-fate/granted/pkg/testable"
	cfflags "github.com/common-fate/granted/pkg/urfav_overrides"
	"github.com/fatih/color"
	"github.com/hako/durafmt"
	"github.com/urfave/cli/v2"
)

// Launchers give a command that we need to run in order to launch a browser, such as
// 'open <URL>' or 'firefox --new-tab <URL'. The returned command is a string slice,
// with each element being an argument. (e.g. []string{"firefox", "--new-tab", "<URL>"})
type Launcher interface {
	LaunchCommand(url string, profile string) []string
}

func AssumeCommand(c *cli.Context) error {
	// assumeFlags allows flags to be passed on either side of the role argument.
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

		var opts []cfaws.LoadProfilesOptsFunc

		profiles, err := cfaws.LoadProfiles(opts...)
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

		//set the session creds using the active role if we have one and the flag is set
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

	// if getConsoleURL is true, we'll use the AWS federated login to retrieve a URL to access the console.
	// depending on how Granted is configured, this is then printed to the terminal or a browser is launched at the URL automatically.
	getConsoleURL := !assumeFlags.Bool("env") && (assumeFlags.Bool("console") || assumeFlags.Bool("active-role") || assumeFlags.String("service") != "" || assumeFlags.Bool("url"))

	if getConsoleURL {
		con := console.AWS{
			Profile: profile.Name,
			Service: assumeFlags.String("service"),
			Region:  region,
		}

		creds, err := profile.AssumeConsole(c.Context, configOpts)
		if err != nil {
			return err
		}

		consoleURL, err := con.URL(creds)
		if err != nil {
			return err
		}

		if cfg.DefaultBrowser == browser.FirefoxKey || cfg.DefaultBrowser == browser.FirefoxStdoutKey {
			// tranform the URL into the Firefox Tab Container format.
			consoleURL = fmt.Sprintf("ext+granted-containers:name=%s&url=%s", profile.Name, url.QueryEscape(consoleURL))
		}

		justPrintURL := assumeFlags.Bool("url") || cfg.DefaultBrowser == browser.StdoutKey || cfg.DefaultBrowser == browser.FirefoxStdoutKey
		if justPrintURL {
			// return the url via stdout through the CLI wrapper script and return early.
			fmt.Print(assumeprint.SafeOutput(consoleURL))
			return nil
		}

		browserPath := cfg.CustomBrowserPath
		if browserPath == "" {
			return errors.New("default browser not configured. run `granted browser set` to configure")
		}

		grantedFolder, err := config.GrantedConfigFolder()
		if err != nil {
			return err
		}

		var l Launcher
		switch cfg.DefaultBrowser {
		case browser.ChromeKey:
			l = launcher.ChromeProfile{
				ExecutablePath: browserPath,
				UserDataPath:   path.Join(grantedFolder, "chromium-profiles", "1"), // held over for backwards compatibility, "1" indicates Chrome profiles
			}
		case browser.BraveKey:
			l = launcher.ChromeProfile{
				ExecutablePath: browserPath,
				UserDataPath:   path.Join(grantedFolder, "chromium-profiles", "2"), // held over for backwards compatibility, "2" indicates Brave profiles
			}
		case browser.EdgeKey:
			l = launcher.ChromeProfile{
				ExecutablePath: browserPath,
				UserDataPath:   path.Join(grantedFolder, "chromium-profiles", "3"), // held over for backwards compatibility, "3" indicates Edge profiles
			}
		case browser.ChromiumKey:
			l = launcher.ChromeProfile{
				ExecutablePath: browserPath,
				UserDataPath:   path.Join(grantedFolder, "chromium-profiles", "4"), // held over for backwards compatibility, "4" indicates Chromium profiles
			}
		case browser.FirefoxKey:
			l = launcher.Firefox{
				ExecutablePath: browserPath,
			}
		default:
			l = launcher.Open{}
		}

		printFlagUsage(con.Region, con.Service)
		fmt.Fprintf(color.Error, "\nOpening a console for %s in your browser...\n", profile.Name)

		// now build the actual command to run - e.g. 'firefox --new-tab <URL>'
		args := l.LaunchCommand(consoleURL, con.Profile)

		cmd, err := forkprocess.New(args...)
		if err != nil {
			return err
		}
		err = cmd.Start()
		if err != nil {
			fmt.Fprintf(color.Error, "Granted was unable to open a browser session automatically: %s", err.Error())
			// allow them to try open the url manually
			alert := color.New(color.Bold, color.FgYellow).SprintFunc()
			fmt.Fprintf(os.Stdout, "\nOpen session manually using the following url:\n")
			fmt.Fprintf(os.Stdout, "\n%s\n", alert("", consoleURL))
			return err
		}
		return nil
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
			if assumeFlags.String("aws-credentials-file") != "" {
				err = cfaws.WriteProfileToCredentialsFile(profile.Name, creds, assumeFlags.String("aws-credentials-file"))
				if err != nil {
					return err
				}
			} else {
				err = cfaws.WriteProfileToDefaultCredentialsFile(profile.Name, creds)
				if err != nil {
					return err
				}
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
	fmt.Print(assumeprint.SafeOutput(""))
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

func printFlagUsage(region, service string) {
	var m []string

	if region == "" {
		m = append(m, "use -r to open a specific region")
	}

	if service == "" {
		m = append(m, "use -s to open a specific service")
	}

	if region == "" || service == "" {
		fmt.Fprintf(color.Error, "\nℹ️  %s (https://docs.commonfate.io/granted/usage/console)\n", strings.Join(m, " or "))
	}
}

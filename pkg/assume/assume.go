package assume

import (
	"fmt"
	"os/exec"
	"strings"

	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/common-fate/clio"
	"github.com/common-fate/granted/pkg/assumeprint"
	cfflags "github.com/common-fate/granted/pkg/urfav_overrides"
	"github.com/urfave/cli/v2"
)

// Launchers give a command that we need to run in order to launch a browser, such as
// 'open <URL>' or 'firefox --new-tab <URL'. The returned command is a string slice,
// with each element being an argument. (e.g. []string{"firefox", "--new-tab", "<URL>"})
type Launcher interface {
	LaunchCommand(url string, profile string) []string
	// UseForkProcess returns true if the launcher implementation should call
	// the forkprocess library.
	//
	// For launchers that use 'open' commands, this should be false,
	// as the forkprocess library causes the following error to appear:
	// 	fork/exec open: no such file or directory
	UseForkProcess() bool
}
type execConfig struct {
	Cmd  string
	Args []string
}

func AssumeCommand(c *cli.Context) error {
	// assumeFlags allows flags to be passed on either side of the role argument.
	// to access flags in this command, use assumeFlags.String("region") etc instead of c.String("region")
	assumeFlags, err := cfflags.New("assumeFlags", GlobalFlags(), c)
	if err != nil {
		return err
	}
	if c.Args().First() == "gcp" {
		gcp := AssumeGCP{
			assumeFlags:   assumeFlags,
			getConsoleURL: assumeFlags.Bool("console"),
		}
		err := gcp.Assume(c)
		if err != nil {
			return err
		}
		return nil
	}

	aws := AssumeAWS{
		ctx:         c,
		assumeFlags: assumeFlags,
	}
	err = aws.Assume()
	if err != nil {
		return err
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

// RunExecCommandWithCreds takes in a command, which may be a program and arguments separated by spaces
// it splits these then runs the command with the credentials as the environment.
// The output of this is returned via the assume script to stdout so it may be processed further by piping
func RunExecCommandWithCreds(creds aws.Credentials, region string, cmd string, args ...string) error {
	fmt.Print(assumeprint.SafeOutput(""))
	c := exec.Command(cmd, args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
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
		clio.Infof("%s ( https://docs.commonfate.io/granted/usage/console )", strings.Join(m, " or "))
	}
}

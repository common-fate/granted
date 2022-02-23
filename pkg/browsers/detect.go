package browsers

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/common-fate/granted/pkg/browsers/detect"
	"github.com/common-fate/granted/pkg/config"
	"github.com/common-fate/granted/pkg/testable"
	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
)

const (
	ChromeKey  string = "CHROME"
	FirefoxKey string = "FIREFOX"
	EdgeKey    string = "EDGE"
	BraveKey   string = "BRAVE"
	DefaultKey string = "DEFAULT"
)

//Checks the config to see if the user has already set up their default browser
func UserHasDefaultBrowser(ctx *cli.Context) (bool, error) {

	//just check the config file for the default browser efield

	conf, err := config.Load()

	if err != nil {
		return false, err
	}

	return conf.DefaultBrowser != "", nil
}

func HandleManualBrowserSelection() (string, error) {
	//didn't find it request manual input

	withStdio := survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)
	in := survey.Select{
		Message: "ℹ️  Select your default browser\n",
		Options: []string{"Chrome", "Brave", "Edge", "Firefox"},
	}
	var selection string
	err := testable.AskOne(&in, &selection, withStdio)
	if err != nil {
		return "", err
	}

	if selection != "" {
		fmt.Fprintf(os.Stderr, "%s\n", selection)
		return selection, nil
	}
	return "", nil
}

//finds out which browser the user has as default
func Find() (string, error) {
	outcome := ""
	ops := runtime.GOOS
	switch ops {
	case "windows":
		// @TODO implement default browser search for windows
		outcome = ""
	case "darwin":
		b, err := detect.HandleOSXBrowserSearch()
		if err != nil {
			return "", err
		}
		outcome = b

	case "linux":
		b, err := detect.HandleLinuxBrowserSearch()
		if err != nil {
			return "", err
		}
		outcome = b

	default:
		fmt.Printf("%s os not supported.\n", ops)
	}

	if outcome == "" {
		fmt.Fprintf(os.Stderr, "ℹ️  Could not find default browser\n")
		return HandleManualBrowserSelection()
	}

	return outcome, nil
}

func GetBrowserName(b string) string {

	if strings.Contains(strings.ToLower(b), "chrome") {
		return ChromeKey
	}
	if strings.Contains(strings.ToLower(b), "brave") {
		return BraveKey
	}
	if strings.Contains(strings.ToLower(b), "edge") {
		return EdgeKey
	}
	if strings.Contains(strings.ToLower(b), "firefox") || strings.Contains(strings.ToLower(b), "mozilla") {
		return FirefoxKey
	}
	return DefaultKey
}

func HandleBrowserWizard(ctx *cli.Context) error {
	withStdio := survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)
	fmt.Fprintf(os.Stderr, "Granted works best with Firefox but also supports Chrome, Brave, and Edge (https://granted.dev/browsers).\n\n")

	browserName, err := Find()
	if err != nil {
		return err
	}
	browserTitle := strings.Title(strings.ToLower(GetBrowserName(browserName)))
	fmt.Fprintf(os.Stderr, "ℹ️  Granted has detected that your default browser is %s.\n", browserTitle)

	in := survey.Select{
		Message: "Use this browser with Granted?\n",
		Options: []string{"Yes", "Choose a different browser"},
	}
	var opt string
	err = testable.AskOne(&in, &opt, withStdio)
	if err != nil {
		return err
	}
	if opt != "Yes" {
		browserName, err = HandleManualBrowserSelection()
		if err != nil {
			return err
		}
		browserTitle = strings.Title(strings.ToLower(GetBrowserName(browserName)))
	}

	if GetBrowserName(browserName) == FirefoxKey {
		err = RunFirefoxExtensionPrompts()
		if err != nil {
			return err
		}
	}

	//save the detected browser as the default
	conf, err := config.Load()
	if err != nil {
		return err
	}

	if conf.DefaultBrowser != browserName {
		conf = &config.Config{DefaultBrowser: GetBrowserName(browserName)}

		err = conf.Save()
		if err != nil {
			return err
		}
	}
	alert := color.New(color.Bold, color.FgGreen).SprintfFunc()

	fmt.Fprintf(os.Stderr, "\n%s\n", alert("✅  Granted will default to using %s.", browserTitle))

	return nil
}

func GrantedIntroduction() {
	fmt.Fprintf(os.Stderr, "\nTo change the web browser that Granted uses run: `granted browser`\n")
	fmt.Fprintf(os.Stderr, "\n\nHere's how to use Granted to supercharge your cloud access:\n")
	fmt.Fprintf(os.Stderr, "\n`assume`                   - search profiles to assume\n")
	fmt.Fprintf(os.Stderr, "\n`assume <PROFILE_NAME>`    - assume a profile\n")
	fmt.Fprintf(os.Stderr, "\n`assume -c <PROFILE_NAME>` - open the console for the specified profile\n")

	os.Exit(0)

}

func RunFirefoxExtensionPrompts() error {
	fmt.Fprintf(os.Stderr, "ℹ️  In order to use Granted with Firefox you need to download the Granted Firefox addon: https://addons.mozilla.org/en-GB/firefox/addon/granted.\nThis addon has minimal permissions and does not access any web page contents (https://granted.dev/firefox-addon).\n")

	label := "\nOpen Firefox to download the extension?\n"

	withStdio := survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)
	in := &survey.Select{
		Message: label,
		Options: []string{"Yes", "Already installed", "No"},
	}
	var out string
	err := testable.AskOne(in, &out, withStdio)
	if err != nil {
		return err
	}

	if out == "No" {
		return errors.New("cancelled browser setup")
	}
	// Allow the user to bypass this step if they have been testing different browsers
	if out == "Already installed" {
		return nil
	}

	opSys := runtime.GOOS
	firefoxPath := ""
	switch opSys {
	case "windows":
		firefoxPath = FirefoxPathWindows
	case "darwin":
		firefoxPath = FirefoxPathMac
	case "linux":
		firefoxPath = FirefoxPathLinux
	default:
		return errors.New("os not supported")
	}

	cmd := exec.Command(firefoxPath,
		"--new-tab",
		"https://addons.mozilla.org/en-GB/firefox/addon/granted/")
	err = cmd.Start()
	if err != nil {
		return err
	}

	// detach from this new process because it continues to run
	err = cmd.Process.Release()
	if err != nil {
		return err
	}
	time.Sleep(time.Second * 2)
	confIn := &survey.Confirm{
		Message: "Type Y to continue once you have installed the extension",
		Default: true,
	}
	var confirm bool
	err = testable.AskOne(confIn, &confirm, withStdio)
	if err != nil {
		return err
	}

	if !confirm {
		return errors.New("cancelled browser setup")
	}
	return nil
}

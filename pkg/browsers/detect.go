package browsers

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/common-fate/granted/pkg/config"
	"github.com/common-fate/granted/pkg/testable"
	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
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
		Message: "Select your default browser",
		Options: []string{"Chrome", "Brave", "Edge", "Firefox", "Chromium"},
	}
	var selection string
	fmt.Fprintln(os.Stderr)
	err := testable.AskOne(&in, &selection, withStdio)
	if err != nil {
		return "", err
	}

	return selection, nil
}

//finds out which browser the user has as default
func Find() (string, error) {
	outcome := ""
	ops := runtime.GOOS
	switch ops {
	case "windows":
		b, err := HandleWindowsBrowserSearch()
		if err != nil {
			return "", err
		}
		outcome = b
	case "darwin":
		b, err := HandleOSXBrowserSearch()
		if err != nil {
			return "", err
		}
		outcome = b

	case "linux":
		b, err := HandleLinuxBrowserSearch()
		if err != nil {
			return "", err
		}
		outcome = b

	default:
		fmt.Printf("%s os not supported.\n", ops)
	}

	if outcome == "" {
		fmt.Fprintf(os.Stderr, "\nℹ️  Could not find default browser\n")
		return HandleManualBrowserSelection()
	}

	return outcome, nil
}

func GetBrowserKey(b string) string {
	if strings.Contains(strings.ToLower(b), "chrome") {
		return ChromeKey
	}

	if strings.Contains(strings.ToLower(b), "chromium") {
		return ChromiumKey
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

// DetectInstallation checks if the default filepath exists for the browser executables on the current os
// returns the detected path
func DetectInstallation(browserKey string) (string, bool) {
	var bPath []string
	switch browserKey {
	case ChromeKey:
		bPath, _ = ChromePathDefaults()
	case BraveKey:
		bPath, _ = BravePathDefaults()
	case EdgeKey:
		bPath, _ = EdgePathDefaults()
	case FirefoxKey:
		bPath, _ = FirefoxPathDefaults()
	case ChromiumKey:
		bPath, _ = ChromiumPathDefaults()
	default:
		return "", false
	}
	if len(bPath) == 0 {
		return "", false
	}
	for _, p := range bPath {
		_, err := os.Stat(p)
		if err == nil {
			return p, true
		}
	}
	return "", false
}

func HandleBrowserWizard(ctx *cli.Context) error {
	withStdio := survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)
	fmt.Fprintf(os.Stderr, "\nGranted works best with Firefox but also supports Chrome, Brave, and Edge (https://granted.dev/browsers).\n")

	browserName, err := Find()
	if err != nil {
		return err
	}
	browserTitle := strings.Title(strings.ToLower(GetBrowserKey(browserName)))
	fmt.Fprintf(os.Stderr, "\nℹ️  Granted has detected that your default browser is %s.\n", browserTitle)

	in := survey.Select{
		Message: "Use this browser with Granted?",
		Options: []string{"Yes", "Choose a different browser"},
	}
	var opt string
	fmt.Fprintln(os.Stderr)
	err = testable.AskOne(&in, &opt, withStdio)
	if err != nil {
		return err
	}
	if opt != "Yes" {
		browserName, err = HandleManualBrowserSelection()
		if err != nil {
			return err
		}
	}

	return ConfigureBrowserSelection(browserName, "")
}

//ConfigureBrowserSelection will verify the existance of the browser executable and promot for a path if it cannot be found
func ConfigureBrowserSelection(browserName string, path string) error {
	browserKey := GetBrowserKey(browserName)
	withStdio := survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)
	browserTitle := strings.Title(strings.ToLower(browserKey))
	// We allow users to configure a custom install path is we cannot detect the installation
	customBrowserPath := path
	// detect installation
	if customBrowserPath != "" {
		_, err := os.Stat(customBrowserPath)
		if err != nil {
			return errors.Wrap(err, "provided path is invalid")
		}
	} else {
		customBrowserPath, detected := DetectInstallation(browserKey)
		if !detected {
			fmt.Fprintf(os.Stderr, "\nℹ️  Granted could not detect an existing installation of %s at known installation paths for your system.\nIf you have already installed this browser, you can specify the path to the executable manually.\n", browserTitle)
			validPath := false
			for !validPath {
				// prompt for custom path
				bpIn := survey.Input{Message: fmt.Sprintf("Please enter the full path to your browser installation for %s:", browserTitle)}
				fmt.Fprintln(os.Stderr)
				err := testable.AskOne(&bpIn, &customBrowserPath, withStdio)
				if err != nil {
					return err
				}
				if _, err := os.Stat(customBrowserPath); err == nil {
					validPath = true
				} else {
					fmt.Fprintf(os.Stderr, "\n❌ The path you entered is not valid\n")
				}
			}
		}
	}

	if browserKey == FirefoxKey {
		err := RunFirefoxExtensionPrompts(customBrowserPath)
		if err != nil {
			return err
		}
	}

	//save the detected browser as the default
	conf, err := config.Load()
	if err != nil {
		return err
	}

	conf.DefaultBrowser = browserKey
	conf.CustomBrowserPath = customBrowserPath
	err = conf.Save()
	if err != nil {
		return err
	}

	alert := color.New(color.Bold, color.FgGreen).SprintfFunc()

	fmt.Fprintf(os.Stderr, "\n%s\n", alert("✅  Granted will default to using %s.", browserTitle))

	return nil
}

func GrantedIntroduction() {
	fmt.Fprintf(os.Stderr, "\nTo change the web browser that Granted uses run: `granted browser -set`\n")
	fmt.Fprintf(os.Stderr, "\n\nHere's how to use Granted to supercharge your cloud access:\n")
	fmt.Fprintf(os.Stderr, "\n`assume`                   - search profiles to assume\n")
	fmt.Fprintf(os.Stderr, "\n`assume <PROFILE_NAME>`    - assume a profile\n")
	fmt.Fprintf(os.Stderr, "\n`assume -c <PROFILE_NAME>` - open the console for the specified profile\n")

	os.Exit(0)

}

func RunFirefoxExtensionPrompts(firefoxPath string) error {
	fmt.Fprintf(os.Stderr, "\nℹ️  In order to use Granted with Firefox you need to download the Granted Firefox addon: https://addons.mozilla.org/en-GB/firefox/addon/granted.\nThis addon has minimal permissions and does not access any web page contents (https://granted.dev/firefox-addon).\n")

	label := "Open Firefox to download the extension?"

	withStdio := survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)
	in := &survey.Select{
		Message: label,
		Options: []string{"Yes", "Already installed", "No"},
	}
	var out string
	fmt.Fprintln(os.Stderr)
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
	fmt.Fprintln(os.Stderr)
	err = testable.AskOne(confIn, &confirm, withStdio)
	if err != nil {
		return err
	}

	if !confirm {
		return errors.New("cancelled browser setup")
	}
	return nil
}

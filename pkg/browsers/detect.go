package browsers

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/common-fate/granted/pkg/config"
	"github.com/common-fate/granted/pkg/testable"
	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
)

const (
	ChromeKey  string = "CHROME"
	FirefoxKey        = "FIREFOX"
	EdgeKey           = "EDGE"
	BraveKey          = "BRAVE"
	DefaultKey        = "DEFAULT"
)

type plist struct {
	//XMLName xml.Name `xml:"plist"`
	Pdict Pdict `xml:"dict"`
}

type Pdict struct {
	//XMLName xml.Name `xml:"dict"`
	Key   string `xml:"key"`
	Array Array  `xml:"array"`
}

type Array struct {
	//XMLName xml.Name `xml:"array"`
	Dict Dict `xml:"dict"`
}

type Dict struct {
	//XMLName xml.Name `xml:"dict"`
	Key     []string `xml:"key"`
	Dict    IntDict  `xml:"dict"`
	Strings []string `xml:"string"`
}

type IntDict struct {
	//XMLName xml.Name `xml:"dict"`
	Key     string `xml:"key"`
	Strings string `xml:"string"`
}

//Checks the config to see if the user has already set up their default browser
func UserHasDefaultBrowser(ctx *cli.Context) (bool, error) {

	//just check the config file for the default browser efield

	conf, err := config.Load()

	if err != nil {
		return false, err
	}

	return conf.DefaultBrowser != "", nil
}

func handleOSXBrowserSearch() (string, error) {
	//get home dir
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	path := home + "/Library/Preferences/com.apple.LaunchServices/com.apple.launchservices.secure.plist"

	//convert plist to xml using putil
	//plutil -convert xml1
	args := []string{"-convert", "xml1", path}
	cmd := exec.Command("plutil", args...)
	err = cmd.Run()
	if err != nil {
		return "", err
	}

	//read plist file
	data, err := ioutil.ReadFile(path)

	if err != nil {
		return "", err
	}
	plist := &plist{}

	// fmt.Fprintf(os.Stderr, "\n%s\n", data)
	//unmarshal the xml into the structs
	err = xml.Unmarshal([]byte(data), &plist)
	if err != nil {
		return "", err
	}

	//get out the default browser

	for i, s := range plist.Pdict.Array.Dict.Strings {
		if s == "http" {
			return plist.Pdict.Array.Dict.Strings[i-1], nil
		}
	}
	return "", nil
}

func handleLinuxBrowserSearch() (string, error) {
	out, err := exec.Command("xdg-settings", "get", "default-web-browser").Output()

	if err != nil {
		return "", err
	}

	return string(out), nil
}

func handleWindowsBrowserSearch() (string, error) {
	//TODO: automatic detection for windows
	outcome, err := HandleManualBrowserSelection()

	if err != nil {
		return "", err
	}

	if outcome != "" {

		conf, err := config.Load()
		if err != nil {
			return "", err
		}

		conf.DefaultBrowser = GetBrowserName(outcome)

		conf.Save()
		alert := color.New(color.Bold, color.FgGreen).SprintFunc()

		fmt.Fprintf(os.Stderr, "\n%s\n", alert("✅  Default browser set."))
	}

	return "", nil
}

func HandleManualBrowserSelection() (string, error) {
	//didn't find it request manual input

	withStdio := survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)
	in := survey.Select{
		Message: "ℹ️  Select your default browser\n",
		Options: []string{"Chrome", "Brave", "Edge", "Firefox"},
	}
	var roleacc string
	err := testable.AskOne(&in, &roleacc, withStdio)
	if err != nil {
		return "", err
	}

	if roleacc != "" {
		fmt.Fprintf(os.Stderr, "%s\n", roleacc)
		return roleacc, nil
	}
	return "", nil
}

//finds out which browser the use has as default
func Find() (string, error) {

	outcome := ""
	// @TODO confirm this works
	ops := runtime.GOOS
	switch ops {
	case "windows":
		b, err := handleWindowsBrowserSearch()
		if err != nil {
			return "", err
		}

		outcome = b
		//return b, nil
	case "darwin":
		b, err := handleOSXBrowserSearch()
		if err != nil {
			return "", err
		}
		outcome = b

	case "linux":
		b, err := handleLinuxBrowserSearch()
		if err != nil {
			return "", err
		}
		outcome = b

	default:
		fmt.Printf("%s.\n", ops)
	}

	if outcome == "" {
		fmt.Fprintf(os.Stderr, "ℹ️  Could not find default browser\n")
		outcome, err := HandleManualBrowserSelection()

		if err != nil {
			return "", err
		}
		return outcome, nil
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

	browserName, err := Find()
	if err != nil {
		return err
	}

	if strings.Contains(strings.ToLower(browserName), "chrome") {
		fmt.Fprintf(os.Stderr, "ℹ️  Granted has detected that your default browser is a Chromium based browser: %s\n", GetBrowserName(browserName))

		withStdio := survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)
		in := survey.Select{
			Message: "Use this browser with Granted?\n",
			Options: []string{"Yes", "Choose a different browser"},
		}
		var opt string
		err := testable.AskOne(&in, &opt, withStdio)
		if err != nil {
			return err
		}

		if opt == "Yes" {
			//save the detected browser as the default
			conf, err := config.Load()
			if err != nil {
				return err
			}

			if conf.DefaultBrowser != browserName {
				conf = &config.Config{DefaultBrowser: GetBrowserName(browserName)}

				conf.Save()
			}
			alert := color.New(color.Bold, color.FgGreen).SprintFunc()

			fmt.Fprintf(os.Stderr, "\n%s\n", alert("✅  Granted will default to using chrome"))

		}
	}

	if strings.Contains(strings.ToLower(browserName), "brave") {
		fmt.Fprintf(os.Stderr, "ℹ️  Granted has detected that your default browser is a Chromium based browser: %s\n", GetBrowserName(browserName))
		withStdio := survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)
		in := survey.Select{
			Message: "Use this browser with Granted?\n",
			Options: []string{"Yes", "Choose a different browser"},
		}
		var opt string
		err := testable.AskOne(&in, &opt, withStdio)
		if err != nil {
			return err
		}

		if opt == "Yes" {
			//save the detected browser as the default
			conf, err := config.Load()
			if err != nil {
				return err
			}

			if conf.DefaultBrowser != browserName {
				conf = &config.Config{DefaultBrowser: GetBrowserName(browserName)}

				conf.Save()
			}
			alert := color.New(color.Bold, color.FgGreen).SprintFunc()

			fmt.Fprintf(os.Stderr, "\n%s\n", alert("✅  Granted will default to using brave."))

		}
	}

	if strings.Contains(strings.ToLower(browserName), "edge") {
		fmt.Fprintf(os.Stderr, "ℹ️  Granted has detected that your default browser is a Chromium based browser: %s\n", GetBrowserName(browserName))

		withStdio := survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)
		in := survey.Select{
			Message: "Use this browser with Granted?\n",
			Options: []string{"Yes", "Choose a different browser"},
		}
		var opt string
		err := testable.AskOne(&in, &opt, withStdio)
		if err != nil {
			return err
		}

		if opt == "Yes" {
			//save the detected browser as the default
			conf, err := config.Load()
			if err != nil {
				return err
			}

			if conf.DefaultBrowser != browserName {
				conf = &config.Config{DefaultBrowser: GetBrowserName(browserName)}

				conf.Save()
			}
			alert := color.New(color.Bold, color.FgGreen).SprintFunc()

			fmt.Fprintf(os.Stderr, "\n%s\n", alert("✅  Granted will default to using edge"))

		}
	}

	if strings.Contains(strings.ToLower(browserName), "firefox") {
		fmt.Fprintf(os.Stderr, "ℹ️  Granted has detected that your default browser is Mozilla Firefox.\n")

		withStdio := survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)
		in := survey.Select{
			Message: "Use this browser with Granted?\n",
			Options: []string{"Yes", "Choose a different browser"},
		}
		var opt string
		err := testable.AskOne(&in, &opt, withStdio)
		if err != nil {
			return err
		}

		if opt == "Yes" {

			err = RunFirefoxExtensionPrompts()
			if err != nil {
				return err
			}
		}
	}

	//if we dont find any automaticly ask for them to select

	conf, err := config.Load()
	if err != nil {
		return err
	}

	if conf.DefaultBrowser == "" {
		outcome, err := HandleManualBrowserSelection()
		if err != nil {
			return err
		}

		conf.DefaultBrowser = GetBrowserName(outcome)

		conf.Save()

		alert := color.New(color.Bold, color.FgGreen).SprintFunc()

		fmt.Fprintf(os.Stderr, "\n%s\n", alert("✅  Granted will default to using ", outcome))

		if strings.Contains(strings.ToLower(outcome), "firefox") {
			err = RunFirefoxExtensionPrompts()

			if err != nil {
				return err
			}
		}

	}

	return nil
}

func GrantedIntroduction() {
	fmt.Fprintf(os.Stderr, "\nTo change default browser in Granted run: `granted browser`\n")
	fmt.Fprintf(os.Stderr, "\n\nHere's how to use Granted to supercharge your cloud access:\n")
	fmt.Fprintf(os.Stderr, "\n`assume`                   - search profiles to assume\n")
	fmt.Fprintf(os.Stderr, "\n`assume <PROFILE_NAME>`    - assume a profile\n")
	fmt.Fprintf(os.Stderr, "\n`assume -c <PROFILE_NAME>` - open the console for the profile\n")

	os.Exit(0)

}

func RunFirefoxExtensionPrompts() error {
	fmt.Fprintf(os.Stderr, "ℹ️  You will need to download and install an extension for firefox to use Granted to its full potential\n")

	label := "\nTake me to download extension?\n"

	withStdio := survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)
	in := &survey.Confirm{
		Message: label,
		Default: true,
	}
	var confirm bool
	err := testable.AskOne(in, &confirm, withStdio)
	if err != nil {
		return err
	}

	if !confirm {
		return errors.New("cancelled browser wizard")
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
	cmd.Process.Release()
	if err != nil {
		return err
	}
	time.Sleep(time.Second * 2)
	alert := color.New(color.Bold, color.FgGreen).SprintFunc()

	conf, err := config.Load()
	if err != nil {
		return err
	}

	if conf.DefaultBrowser != "firefox" {
		conf = &config.Config{DefaultBrowser: GetBrowserName("firefox")}

		conf.Save()
	}

	fmt.Fprintf(os.Stderr, "\n%s\n", alert("✅  Granted will default to using firefox"))

	return nil
}

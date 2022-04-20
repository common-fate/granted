package granted

import (
	"fmt"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/common-fate/granted/pkg/browsers"
	"github.com/common-fate/granted/pkg/config"
	"github.com/common-fate/granted/pkg/testable"
	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var DefaultBrowserCommand = cli.Command{
	Name:        "browser",
	Usage:       "View the web browser that Granted uses to open cloud consoles",
	Subcommands: []*cli.Command{&SetBrowserCommand, &SetSSOBrowserCommand},
	Action: func(c *cli.Context) error {
		//return the default browser that is set
		conf, err := config.Load()
		if err != nil {
			return err
		}
		fmt.Fprintf(color.Error, "Granted is using %s. To change this run `granted browser set`.\n", conf.DefaultBrowser)

		return nil
	},
}

var SetBrowserCommand = cli.Command{
	Name:  "set",
	Usage: "Change the web browser that Granted uses to open cloud consoles",
	Flags: []cli.Flag{&cli.StringFlag{Name: "browser", Aliases: []string{"b"}, Usage: "Specify a default browser without prompts, e.g `-b firefox`, `-b chrome`"},
		&cli.StringFlag{Name: "path", Aliases: []string{"p"}, Usage: "Specify a path to the browser without prompts, requires -browser to be provided"}},
	Action: func(c *cli.Context) (err error) {
		outcome := c.String("browser")
		path := c.String("path")

		if outcome == "" {
			if path != "" {
				fmt.Fprintln(color.Error, "-path flag must be used with -browser flag, provided path will be ignored.")
			}
			outcome, err = browsers.HandleManualBrowserSelection()
			if err != nil {
				return err
			}
		}

		return browsers.ConfigureBrowserSelection(outcome, path)
	},
}

var SetSSOBrowserCommand = cli.Command{
	Name:  "set-sso",
	Usage: "Change the web browser that Granted uses to sso flows",
	Flags: []cli.Flag{&cli.StringFlag{Name: "browser", Aliases: []string{"b"}, Usage: "Specify a default browser without prompts, e.g `-b firefox`, `-b chrome`"},
		&cli.StringFlag{Name: "path", Aliases: []string{"p"}, Usage: "Specify a path to the browser without prompts, requires -browser to be provided"}},
	Action: func(c *cli.Context) (err error) {
		outcome := c.String("browser")
		path := c.String("path")

		if outcome == "" {
			if path != "" {
				fmt.Fprintln(color.Error, "-path flag must be used with -browser flag, provided path will be ignored.")
			}
			fmt.Fprintf(color.Error, "\nℹ️  Select your SSO default browser\n")
			outcome, err = browsers.HandleManualBrowserSelection()
			if err != nil {
				return err
			}

			//save the detected browser as the default
			conf, err := config.Load()
			if err != nil {
				return err
			}

			browserKey := browsers.GetBrowserKey(outcome)
			withStdio := survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)
			title := cases.Title(language.AmericanEnglish)
			browserTitle := title.String(strings.ToLower(browserKey))
			// We allow users to configure a custom install path is we cannot detect the installation
			browserPath := ""
			// detect installation
			if browserKey != browsers.FirefoxStdoutKey && browserKey != browsers.StdoutKey {

				customBrowserPath, detected := browsers.DetectInstallation(browserKey)
				if !detected {
					fmt.Fprintf(color.Error, "\nℹ️  Granted could not detect an existing installation of %s at known installation paths for your system.\nIf you have already installed this browser, you can specify the path to the executable manually.\n", browserTitle)
					validPath := false
					for !validPath {
						// prompt for custom path
						bpIn := survey.Input{Message: fmt.Sprintf("Please enter the full path to your browser installation for %s:", browserTitle)}
						fmt.Fprintln(color.Error)
						err := testable.AskOne(&bpIn, &customBrowserPath, withStdio)
						if err != nil {
							return err
						}
						if _, err := os.Stat(customBrowserPath); err == nil {
							validPath = true
						} else {
							fmt.Fprintf(color.Error, "\n❌ The path you entered is not valid\n")
						}
					}
				}
				browserPath = customBrowserPath

			}

			conf.CustomSSOBrowserPath = browserPath
			err = conf.Save()
			if err != nil {
				return err
			}

			alert := color.New(color.Bold, color.FgGreen).SprintfFunc()

			fmt.Fprintf(color.Error, "\n%s\n", alert("✅  Granted will default to using %s for SSO flows.", browserKey))
		}
		return nil
	},
}

package granted

import (
	"fmt"
	"os"
	"strings"

	"github.com/common-fate/granted/pkg/browsers"
	"github.com/common-fate/granted/pkg/config"
	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
)

var DefaultBrowserCommand = cli.Command{
	Name:  "browser",
	Usage: "Change the web browser that Granted uses to open cloud consoles",
	Flags: []cli.Flag{
		&cli.BoolFlag{Name: "set", Aliases: []string{"s"}, Usage: "Set browser name"},
	},
	Action: func(c *cli.Context) error {

		if c.Bool("set") {
			outcome, err := browsers.HandleManualBrowserSelection()
			if err != nil {
				return err
			}

			if outcome != "" {
				conf, err := config.Load()
				if err != nil {
					return err
				}

				conf.DefaultBrowser = browsers.GetBrowserName(outcome)

				if strings.Contains(strings.ToLower(outcome), "firefox") {
					err = browsers.RunFirefoxExtensionPrompts()

					if err != nil {
						return err
					}
				}

				err = conf.Save()
				if err != nil {
					return err
				}
				alert := color.New(color.Bold, color.FgGreen).SprintFunc()

				fmt.Fprintf(os.Stderr, "\n%s\n", alert("✅  Granted web browser set."))
			}
			return nil

		} else {
			//return the default browser that is set
			conf, err := config.Load()
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Default: %s\n", conf.DefaultBrowser)

		}
		return nil

	},
}

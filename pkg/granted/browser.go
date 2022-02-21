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
	Usage: "Update your default browser",
	Flags: []cli.Flag{
		&cli.BoolFlag{Name: "set", Aliases: []string{"s"}, Usage: "Set browser name"},
	},
	Action: func(c *cli.Context) error {
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

			conf.Save()
			alert := color.New(color.Bold, color.FgGreen).SprintFunc()

			fmt.Fprintf(os.Stderr, "\n%s\n", alert("✅  Default browser set."))
		}
		return nil
	},
}

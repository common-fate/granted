package granted

import (
	"fmt"
	"os"

	"github.com/common-fate/granted/pkg/browsers"
	"github.com/common-fate/granted/pkg/config"
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

			return browsers.ConfigureBrowserSelection(outcome)

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

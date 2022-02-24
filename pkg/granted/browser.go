package granted

import (
	"fmt"
	"os"

	"github.com/common-fate/granted/pkg/browsers"
	"github.com/common-fate/granted/pkg/config"
	"github.com/urfave/cli/v2"
)

var DefaultBrowserCommand = cli.Command{
	Name:        "browser",
	Usage:       "View the web browser that Granted uses to open cloud consoles",
	Subcommands: []*cli.Command{&SetBrowserCommand},
	Action: func(c *cli.Context) error {
		//return the default browser that is set
		conf, err := config.Load()
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Granted is using %s. To change this run `granted browser set`.\n", conf.DefaultBrowser)

		return nil
	},
}

var SetBrowserCommand = cli.Command{
	Name:  "set",
	Usage: "Change the web browser that Granted uses to open cloud consoles",
	Action: func(c *cli.Context) error {
		outcome, err := browsers.HandleManualBrowserSelection()
		if err != nil {
			return err
		}

		return browsers.ConfigureBrowserSelection(outcome)
	},
}

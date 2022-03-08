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
	Flags: []cli.Flag{&cli.StringFlag{Name: "browser", Aliases: []string{"b"}, Usage: "Specify a default browser without prompts, e.g `-b firefox`, `-b chrome`"},
		&cli.StringFlag{Name: "path", Aliases: []string{"p"}, Usage: "Specify a path to the browser without prompts, requires -browser to be provided"}},
	Action: func(c *cli.Context) (err error) {
		outcome := c.String("browser")
		path := c.String("path")

		if outcome == "" {
			if path != "" {
				fmt.Fprintln(os.Stderr, "-path flag must be usedwith -browser flag, provided path will be ignored.")
			}
			outcome, err = browsers.HandleManualBrowserSelection()
			if err != nil {
				return err
			}
		}

		return browsers.ConfigureBrowserSelection(outcome, path)
	},
}

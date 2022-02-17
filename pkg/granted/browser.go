package granted

import (
	"fmt"
	"os"

	"github.com/common-fate/granted/pkg/assume"
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
		//ctx := c.Context
		outcome, err := assume.HandleManualBrowserSelection()

		if err != nil {
			return err
		}

		if outcome != "" {

			conf, err := config.Load()
			if err != nil {
				return err
			}

			conf = &config.Config{DefaultBrowser: assume.GetBrowserName(outcome)}

			conf.Save()
			alert := color.New(color.Bold, color.FgGreen).SprintFunc()

			fmt.Fprintf(os.Stderr, "\n%s\n", alert("✅  Set %s as default browser", outcome))
		}
		return nil
	},
}

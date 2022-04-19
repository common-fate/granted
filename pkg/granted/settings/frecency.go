package settings

import (
	"fmt"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/common-fate/granted/pkg/config"
	"github.com/common-fate/granted/pkg/testable"
	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
)

var SetProfileOrderingCommand = cli.Command{
	Name:  "set-order",
	Usage: "Update profile ordering when assuming",
	Action: func(c *cli.Context) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		withStdio := survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)
		in := survey.Select{
			Message: "Select filter type",
			Options: []string{"Frecency", "Alphabetical"},
		}
		var selection string
		fmt.Fprintln(color.Error)
		err = testable.AskOne(&in, &selection, withStdio)
		if err != nil {
			return err
		}

		cfg.Ordering = selection
		err = cfg.Save()
		if err != nil {
			return err
		}

		green := color.New(color.FgGreen)

		green.Fprintln(color.Error, "Set profile ordering to: ", selection)
		return nil

	},
}

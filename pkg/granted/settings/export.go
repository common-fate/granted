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

var ExportSettingsCommand = cli.Command{
	Name:        "export-suffix",
	Usage:       "sets a exported profile name with a user specified suffix",
	Subcommands: []*cli.Command{&SetExportSettingsCommand},
	Action: func(c *cli.Context) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		green := color.New(color.FgGreen)

		green.Fprintln(color.Error, "Current suffix: ", cfg.ExportCredentialSuffix)
		return nil

	},
}

var SetExportSettingsCommand = cli.Command{
	Name:  "set",
	Usage: "Sets the prefix",
	Action: func(c *cli.Context) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		withStdio := survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)
		in := survey.Input{
			Message: "Exported credential suffix",
		}
		var selection string
		fmt.Fprintln(color.Error)
		err = testable.AskOne(&in, &selection, withStdio)
		if err != nil {
			return err
		}

		cfg.ExportCredentialSuffix = selection
		err = cfg.Save()
		if err != nil {
			return err
		}

		green := color.New(color.FgGreen)

		green.Fprintln(color.Error, "Set export credential suffix to: ", selection)
		return nil

	},
}

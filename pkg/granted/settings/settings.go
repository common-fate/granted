package settings

import (
	"github.com/urfave/cli/v2"
)

var SettingsCommand = cli.Command{
	Name:        "settings",
	Usage:       "Manage Granted settings",
	Subcommands: []*cli.Command{&PrintCommand, &ProfileOrderingCommand, &ExportSettingsCommand, &RequestURLCommand},
	Action:      PrintCommand.Action,
}

package settings

import (
	"github.com/urfave/cli/v2"

	"github.com/common-fate/granted/pkg/granted/settings/requesturl"
)

var SettingsCommand = cli.Command{
	Name:        "settings",
	Usage:       "Manage Granted settings",
	Subcommands: []*cli.Command{&PrintCommand, &ProfileOrderingCommand, &ExportSettingsCommand, &requesturl.Commands, &SetConfigCommand},
	Action:      PrintCommand.Action,
}

package commonfate

import (
	"github.com/common-fate/cli/cmd/cli/command"
	"github.com/urfave/cli/v2"
)

var CommonFateCommand = cli.Command{
	Name:        "commonfate",
	Usage:       "Interact with a Common Fate deployment",
	Flags:       []cli.Flag{},
	Subcommands: []*cli.Command{&command.Configure, &command.Login, &command.Context},
}

package alias

import (
	"os"

	"github.com/urfave/cli/v2"
)

// Require takes a urfave/cli command
// and wraps it's Action in "middleware"
// requiring the Granted alias to be set up.
func Require(cmd *cli.Command) *cli.Command {
	original := cmd.Action
	cmd.Action = func(c *cli.Context) error {
		if os.Getenv("FORCE_NO_ALIAS") != "true" {
			err := MustBeConfigured()
			if err != nil {
				return err
			}
		}
		return original(c)
	}
	return cmd
}

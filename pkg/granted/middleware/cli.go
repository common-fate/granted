package middleware

import "github.com/urfave/cli/v2"

func ShouldShowHelp(c *cli.Context) bool {
	args := c.Args().Slice()
	// if help argument is provided then skip this check.
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			return true
		}
	}
	return false
}
func WithBeforeFuncs(cmd *cli.Command, funcs ...cli.BeforeFunc) *cli.Command {
	// run the commands own before function last if it exists
	// this will help to ensure we have meaningful levels of error precedence
	// e.g check if deployment config exists before checking for aws credentials
	b := cmd.Before
	cmd.Before = func(c *cli.Context) error {
		// skip before funcs and allows the help to be displayed
		if ShouldShowHelp(c) {
			return nil
		}
		for _, f := range funcs {
			err := f(c)
			if err != nil {
				return err
			}
		}
		if b != nil {
			err := b(c)
			if err != nil {
				return err
			}
		}
		return nil
	}
	return cmd
}

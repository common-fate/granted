package granted

import (
	"fmt"
	"os"

	"github.com/common-fate/granted/pkg/config"
	"github.com/common-fate/granted/pkg/credstore"
	"github.com/urfave/cli/v2"
)

var DefaultTokenCommand = cli.Command{
	Name:        "token",
	Usage:       "Functionality to make ",
	Subcommands: []*cli.Command{&TokenListCommand, &ClearTokensCommand},
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

var TokenListCommand = cli.Command{
	Name:  "list",
	Usage: "Remove all saved tokens from keyring and delete all granted configuration",
	Action: func(ctx *cli.Context) error {
		tokens, err := credstore.List()
		if err != nil {
			return err
		}
		for i, token := range tokens {
			fmt.Fprintf(os.Stderr, "%d. %s\n", i+1, token)
		}
		return nil
	},
}
var ClearTokensCommand = cli.Command{
	Name:  "clear",
	Usage: "Remove all saved tokens from keyring",
	Action: func(c *cli.Context) error {
		err := credstore.ClearAll()
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Cleared all saved tokens")
		return nil
	},
}

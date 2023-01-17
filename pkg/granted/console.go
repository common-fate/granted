package granted

import (
	"fmt"

	"github.com/common-fate/granted/pkg/cfaws"
	"github.com/common-fate/granted/pkg/console"
	"github.com/urfave/cli/v2"
)

var ConsoleCommand = cli.Command{
	Name:  "console",
	Usage: "Generate an AWS console URL using credentials in the environment",
	Flags: []cli.Flag{&cli.StringFlag{Name: "service"}, &cli.StringFlag{Name: "region", EnvVars: []string{"AWS_REGION"}}},
	Action: func(c *cli.Context) error {
		ctx := c.Context
		credentials := cfaws.GetEnvCredentials(ctx)
		con := console.AWS{
			Service: c.String("service"),
			Region:  c.String("region"),
		}

		consoleURL, err := con.URL(credentials)
		if err != nil {
			return err
		}
		fmt.Println(string(consoleURL))
		return nil
	},
}

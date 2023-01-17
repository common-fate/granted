package granted

import (
	"fmt"
	"net/url"

	"github.com/common-fate/granted/pkg/cfaws"
	"github.com/common-fate/granted/pkg/console"
	"github.com/urfave/cli/v2"
)

var ConsoleCommand = cli.Command{
	Name:  "console",
	Usage: "Generate an AWS console URL using credentials in the environment",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "service"},
		&cli.StringFlag{Name: "region", EnvVars: []string{"AWS_REGION"}},
		&cli.BoolFlag{Name: "firefox"},
		&cli.StringFlag{Name: "color", Usage: "When the firefox flag is true, this specifies the color of the container tab"},
		&cli.StringFlag{Name: "icon", Usage: "When firefox flag is true, this specifies the icon of the container tab"},
		&cli.StringFlag{Name: "container-name", Usage: "When firefox flag is true, this specifies the name of the container of the container tab.", Value: "aws"},
	},
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
		if c.Bool("firefox") {
			// tranform the URL into the Firefox Tab Container format.
			consoleURL = fmt.Sprintf("ext+granted-containers:name=%s&url=%s&color=%s&icon=%s", c.String("container-name"), url.QueryEscape(consoleURL), c.String("color"), c.String("icon"))
		}
		fmt.Println(string(consoleURL))
		return nil
	},
}

package cfflags

import (
	"fmt"
	"os"
	"testing"

	"github.com/fatih/color"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v2"
)

var testingFlags = []cli.Flag{
	&cli.BoolFlag{Name: "testbool", Aliases: []string{"c"}, Usage: "Open a web console to the role"},
	&cli.StringFlag{Name: "teststringservice", Aliases: []string{"s"}, Usage: "Specify a service to open the console into"},
	&cli.StringFlag{Name: "teststringregion", Aliases: []string{"r"}, Usage: "Specify a service to open the console into"},
}

func TestFlagsPassToCFFlags(t *testing.T) {

	app := cli.App{
		Name: "test",

		Flags: testingFlags,

		Action: func(c *cli.Context) error {

			assumeFlags, err := New("assumeFlags", testingFlags, c)
			if err != nil {
				return err
			}

			booloutcome := assumeFlags.Bool("testbool")

			serviceoutcome := assumeFlags.String("teststringservice")
			regionoutcome := assumeFlags.String("teststringregion")

			assert.Equal(t, booloutcome, true)
			assert.Equal(t, serviceoutcome, "iam")
			assert.Equal(t, regionoutcome, "region-name")
			return nil
		},
		EnableBashCompletion: true,
	}

	os.Args = []string{"", "test-role", "-c", "-s", "iam", "-r", "region-name"}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Fprintf(color.Error, "%s\n", err)
		os.Exit(1)
	}

}

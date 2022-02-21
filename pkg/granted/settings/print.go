package settings

import (
	"os"

	"github.com/common-fate/granted/pkg/debug"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli/v2"
)

var PrintCommand = cli.Command{
	Name:  "print",
	Usage: "List Granted Settings",
	Action: func(c *cli.Context) error {

		data := [][]string{
			{"logging verbosity", debug.CliVerbosity.String()},
			{"update-checker-api-url", c.String("update-checker-api-url")},
		}

		table := tablewriter.NewWriter(os.Stderr)
		table.SetHeader([]string{"SETTING", "VALUE"})
		table.SetAutoWrapText(false)
		table.SetAutoFormatHeaders(true)
		table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
		table.SetAlignment(tablewriter.ALIGN_LEFT)
		table.SetCenterSeparator("")
		table.SetColumnSeparator("")
		table.SetRowSeparator("")
		table.SetRowLine(true)
		table.SetHeaderLine(false)
		table.SetBorder(false)
		table.SetTablePadding("\t")
		table.SetNoWhiteSpace(true)
		table.AppendBulk(data)
		table.Render()
		return nil
	},
}

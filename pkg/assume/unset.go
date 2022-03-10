package assume

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
)

func UnsetAction(c *cli.Context) error {
	//interacts with scripts to unset all the aws environment variables
	fmt.Print("GrantedDesume")
	green := color.New(color.FgGreen)
	green.Fprintf(color.Error, "\nSession credentials revoked\n\n")
	return nil
}

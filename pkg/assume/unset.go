package assume

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
)

func UnsetAction(c *cli.Context) error {
	//interacts with scripts to unset all the aws environment variables
	fmt.Print("GrantedDesume")
	green := color.New(color.FgGreen)
	green.Fprintf(os.Stderr, "\nEnvironment variables cleared\n\n")
	return nil
}

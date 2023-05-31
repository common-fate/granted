package assume

import (
	"fmt"

	"github.com/common-fate/clio"
	"github.com/urfave/cli/v2"
)

func UnsetAction(c *cli.Context) error {
	clio.Success("Environment variables cleared")
	// interacts with scripts to unset all the aws environment variables
	fmt.Print("GrantedDesume")
	return nil
}

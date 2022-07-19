package assume

import (
	"fmt"
	"os"
	"strings"

	"github.com/common-fate/granted/pkg/cfaws"
	"github.com/urfave/cli/v2"
)

// If there are more than 2 args and the last argument is a "-" then provide completion for the flags.
//
// Else, provide completion for the aws profiles.
//
// You can use `assume -c ` + tab to get profile names or `assume -` + tab to get flags
func Completion(ctx *cli.Context) {
	if len(os.Args) > 2 && strings.HasPrefix(os.Args[len(os.Args)-2], "-") {
		// set the ooutput back to std out so that this completion works correctly
		ctx.App.Writer = os.Stdout
		cli.DefaultAppComplete(ctx)
	} else {
		// Ignore errors from this function. Tab completion handles graceful degradation back to listing files.
		awsProfiles, _ := cfaws.LoadProfiles()
		// Tab completion script requires each option to be separated by a newline
		fmt.Println(strings.Join(awsProfiles.ProfileNames, "\n"))
	}
}

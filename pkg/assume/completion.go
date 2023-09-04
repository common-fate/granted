package assume

import (
	"fmt"
	"os"
	"strings"

	"github.com/common-fate/clio"
	"github.com/common-fate/granted/pkg/cfaws"
	"github.com/common-fate/granted/pkg/console"
	"github.com/urfave/cli/v2"
)

// If there are more than 2 args and the last argument is a "-" then provide completion for the flags.
//
// Else, provide completion for the aws profiles.
//
// You can use `assume -c ` + tab to get profile names or `assume -` + tab to get flags
func Completion(ctx *cli.Context) {
	clio.SetLevelFromEnv("GRANTED_LOG")
	if ctx.Bool("verbose") {
		clio.SetLevelFromString("debug")
	}

	// autocompletion for service flag
	if len(os.Args) > 2 {
		if os.Args[len(os.Args)-2] == "-s" || os.Args[len(os.Args)-2] == "-service" {
			for k := range console.ServiceMap {
				fmt.Println(k)
			}
			return
		}

	}

	awsProfiles, _ := cfaws.LoadProfilesFromDefaultFiles()

	// profileName argument can have any position to the command
	// this check will help in not showing the profile name again it's already included.
	hasProfileNameArg := false
	for _, arg := range os.Args {
		for _, awsProfile := range awsProfiles.ProfileNames {
			if arg == awsProfile {
				hasProfileNameArg = true
			}
		}
	}

	if !hasProfileNameArg {
		// Tab completion script requires each option to be separated by a newline
		fmt.Println(strings.Join(awsProfiles.ProfileNames, "\n"))

		return
	}

	// else set the output back to std out so that this completion works correctly
	ctx.App.Writer = os.Stdout
	cli.DefaultAppComplete(ctx)
}

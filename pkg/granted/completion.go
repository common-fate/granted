package granted

import (
	"fmt"
	"os"
	"os/user"

	"github.com/common-fate/granted/internal/build"
	"github.com/common-fate/granted/pkg/assume"
	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
)

var flags = []cli.Flag{
	&cli.StringFlag{
		Name:     "shell",
		Value:    "fish",
		Aliases:  []string{"s"},
		Usage:    "Shell type to generate completion for (fish)",
		Required: true,
	},
}

var CompletionCommand = cli.Command{
	Name:  "completion",
	Usage: "Add autocomplete to your granted cli installation",
	Flags: flags,
	Action: func(c *cli.Context) error {
		// check shell type from flag
		if c.String("shell") == "fish" {

			assumeApp := assume.GetCliApp()

			// Run the native FishCompletion method and generate a string of its outputs
			if build.Version == "dev" {
				fmt.Printf("⚙️  Generating commands for dgranted/dassume\n")
				c.App.Name = "dgranted"
				assumeApp.Name = "dassume"
			} else {
				c.App.Name = "granted"
				assumeApp.Name = "assume"
			}

			grantedAppOutput, _ := c.App.ToFishCompletion()
			assumeAppOutput, _ := assumeApp.ToFishCompletion()
			combinedOutput := fmt.Sprintf("%s\n%s", grantedAppOutput, assumeAppOutput)

			// try fetch user home dir
			user, _ := user.Current()

			executableDir := user.HomeDir + "/.config/fish/completions/granted_completer_fish.fish"

			// Try create a file
			err := os.WriteFile(executableDir, []byte(combinedOutput), 0600)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Something went wrong when saving fish autocompletions: "+err.Error())
			}

			green := color.New(color.FgGreen)

			green.Fprintln(os.Stderr, "[✔] Fish autocompletions generated successfully ")
			fmt.Fprintln(os.Stderr, "To use these completions please run the executable:")
			fmt.Fprintln(os.Stderr, "source "+executableDir)

		} else {
			fmt.Fprintln(os.Stderr, "To install completions for other shells like zsh, bash, please see our docs:")
			fmt.Fprintln(os.Stderr, "https://granted.dev/autocompletion")
			/*
				@TODO: consider adding automatic support for other shells in this same CLI command
					Can be modelled off these tools
					https://github.com/cli/cli/blob/trunk/pkg/cmd/completion/completion.go
					https://github.com/spf13/cobra/blob/master/shell_completions.md
			*/
		}

		return nil
	},

	Description: "To install completions for other shells like zsh, bash, please see our docs:\nhttps://granted.dev/autocompletion\n",

}

package assume

import (
	"fmt"
	"os"
	"os/user"

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
	&cli.StringFlag{
		Name:    "development",
		Value:   "false",
		Aliases: []string{"d"},
		Usage:   "When passed this will compile the autocompletions for dev (dgranted alias)",
	},
}

var CompletionCommand = cli.Command{
	Name:  "completion",
	Usage: "Add autocomplete to your granted cli installation",
	Flags: flags,
	Action: func(c *cli.Context) error {
		// check shell type from flag
		if c.String("shell") == "fish" {

			// Run the native FishCompletion method and generate a string of its outputs
			// If in dev alias the app to *dgranted
			if c.String("development") == "true" {
				fmt.Println("Using dgranted alias for command")
				c.App.Name = "dgranted"
			}
			output, _ := c.App.ToFishCompletion()
			c.App.Name = "granted"

			// try fetch user home dir
			user, _ := user.Current()

			executableDir := user.HomeDir + "/.config/fish/completions/granted_completer_fish.fish"

			// Try create a file
			f, err := os.Create(executableDir)

			if err != nil {
				fmt.Println("Something went wrong when saving fish autocompletions" + err.Error())
			}

			// Defer closing the file
			defer f.Close()
			// Write the string to the file
			_, err = f.WriteString(output)
			if err != nil {
				fmt.Println("Something went wrong when writing fish autocompletions to file")
			}
			f.Close()

			green := color.New(color.FgGreen)

			green.Println("[✔] Fish autocompletions generated successfully ")
			fmt.Println("To use these completions please run the executable:")
			fmt.Println("source " + executableDir)

		} else {
			fmt.Println("To install completions for other shells like zsh, bash, please see our docs:")
			fmt.Println("https://granted.dev/docs/cli/completion")
			/*
				@TODO: consider adding automatic support for other shells in this same CLI command
					Can be modelled off these tools
					https://github.com/cli/cli/blob/trunk/pkg/cmd/completion/completion.go
					https://github.com/spf13/cobra/blob/master/shell_completions.md
			*/
		}

		return nil
	},
}

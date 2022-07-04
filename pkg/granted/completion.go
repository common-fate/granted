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
	Action: func(c *cli.Context) (err error) {
		// check shell type from flag
		if c.String("shell") == "fish" {
			err = installFishCompletions(c)
		} else if c.String("shell") == "zsh" {
			err = installZSHCompletions(c)
		} else if c.String("shell") == "bash" {
			err = installBashCompletions(c)
		} else {
			fmt.Fprintln(color.Error, "To install completions for other shells, please see our docs:")
			fmt.Fprintln(color.Error, "https://granted.dev/autocompletion")
		}
		return err
	},

	Description: "Install completions for fish, zsh, or bash. To install completions for other shells, please see our docs:\nhttps://granted.dev/autocompletion\n",
}

func installFishCompletions(c *cli.Context) error {
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
		return fmt.Errorf("Something went wrong when saving fish autocompletions: " + err.Error())
	}

	green := color.New(color.FgGreen)

	green.Fprintln(color.Error, "[✔] Fish autocompletions generated successfully ")
	fmt.Fprintln(color.Error, "To use these completions please run the executable:")
	fmt.Fprintln(color.Error, "source "+executableDir)
	return nil
}

func installZSHCompletions(c *cli.Context) error {
	// @TODO work out how best to manage installing autocomplete for users
	return nil
}

func installBashCompletions(c *cli.Context) error {
	// @TODO work out how best to manage installing autocomplete for users
	return nil
}

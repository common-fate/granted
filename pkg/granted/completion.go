package granted

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"os"
	"os/user"
	"path"

	"github.com/common-fate/granted/internal/build"
	"github.com/common-fate/granted/pkg/assume"
	"github.com/common-fate/granted/pkg/config"
	"github.com/common-fate/granted/pkg/shells"
	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
)

//go:embed templates
var templateFiles embed.FS
var flags = []cli.Flag{
	&cli.StringFlag{
		Name:     "shell",
		Aliases:  []string{"s"},
		Usage:    "Shell to install completions for (fish, zsh, bash)",
		Required: true,
	},
}

var CompletionCommand = cli.Command{
	Name:  "completion",
	Usage: "Add autocomplete to your granted cli installation",
	Flags: flags,
	Action: func(c *cli.Context) (err error) {
		shell := c.String("shell")
		switch shell {
		case "fish":
			err = installFishCompletions(c)
		case "zsh":
			err = installZSHCompletions(c)
		case "bash":
			err = installBashCompletions(c)
		default:
			fmt.Fprintln(color.Error, "To install completions for other shells, please see our docs:")
			fmt.Fprintln(color.Error, "https://granted.dev/autocompletion")
		}
		return err
	},

	Description: "Install completions for fish, zsh, or bash. To install completions for other shells, please see our docs:\nhttps://granted.dev/autocompletion\n",
}

func installFishCompletions(c *cli.Context) error {
	assumeApp := assume.GetCliApp()
	c.App.Name = build.GrantedBinaryName()
	assumeApp.Name = build.AssumeScriptName()
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

type AutoCompleteTemplateData struct {
	Program string
}

func installZSHCompletions(c *cli.Context) error {
	file, err := shells.GetZshConfigFile()
	if err != nil {
		return err
	}

	tmpl, err := template.ParseFS(templateFiles, "templates/*")
	if err != nil {
		return err
	}

	assumeData := AutoCompleteTemplateData{
		Program: build.AssumeScriptName(),
	}
	assume := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(assume, "zsh_autocomplete_assume.tmpl", assumeData)
	if err != nil {
		return err
	}
	grantedData := AutoCompleteTemplateData{
		Program: build.GrantedBinaryName(),
	}
	granted := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(granted, "zsh_autocomplete_granted.tmpl", grantedData)
	if err != nil {
		return err
	}

	zshPathAssume, err := config.SetupZSHAutoCompleteFolderAssume()
	if err != nil {
		return err
	}

	err = os.WriteFile(path.Join(zshPathAssume, "_"+assumeData.Program), assume.Bytes(), 0666)
	if err != nil {
		return err
	}
	zshPathGranted, err := config.SetupZSHAutoCompleteFolderGranted()
	if err != nil {
		return err
	}
	err = os.WriteFile(path.Join(zshPathGranted, "_"+grantedData.Program), granted.Bytes(), 0666)
	if err != nil {
		return err
	}
	err = shells.AppendLine(file, fmt.Sprintf("fpath=(%s/ $fpath)", zshPathAssume))
	var lae *shells.ErrLineAlreadyExists
	if is := errors.As(err, &lae); !is {
		return err
	}
	err = shells.AppendLine(file, fmt.Sprintf("fpath=(%s/ $fpath)", zshPathGranted))
	lae = nil
	if is := errors.As(err, &lae); !is {
		return err
	}
	green := color.New(color.FgGreen)
	green.Fprintln(color.Error, "[✔] ZSH autocompletions generated successfully ")
	return nil
}

func installBashCompletions(c *cli.Context) error {
	fmt.Fprintln(color.Error, "We don't have completion support for bash yet, check out our docs to find out how to let us know you want this feature.")
	fmt.Fprintln(color.Error, "https://granted.dev/autocompletion")
	return nil
}

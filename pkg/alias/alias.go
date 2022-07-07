// Package alias sets up the shell alias for Granted.
// The alias is required so that the Granted wrapper script
// (scripts/granted in this repository) is sourced rather than executed.
// By sourcing the wrapper script we can export environment variables into
// the user's shell after they call the Granted CLI.
// These variables are typically used for cloud provider session credentials.
package alias

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/common-fate/granted/internal/build"
	"github.com/common-fate/granted/pkg/shells"
	"github.com/fatih/color"
)

func init() {
	// By default the shell wrapper makes the color library
	// think that the terminal doesn't support colors.
	// We override this behaviour here so that we can print colored output.
	// Users can set NO_COLOR to true if they are working in a terminal without
	// color support and want to use Granted there.
	_, color.NoColor = os.LookupEnv("NO_COLOR")
}

const fishAlias = `alias assume="source /usr/local/bin/assume.fish"`
const defaultAlias = `alias assume="source assume"`
const devFishAlias = `alias dassume="source /usr/local/bin/dassume.fish"`
const devDefaultAlias = `alias dassume="source dassume"`

func GetDefaultAlias() string {
	if build.IsDev() {
		return devDefaultAlias
	}
	return defaultAlias
}
func GetFishAlias() string {
	if build.IsDev() {
		return devFishAlias
	}
	return fishAlias
}

type Config struct {
	File  string
	Alias string
}

// IsConfigured returns whether the shell alias is correctly set up
// for Granted.
func IsConfigured() bool {
	return os.Getenv("GRANTED_ALIAS_CONFIGURED") == "true"
}

// MustBeConfigured displays a helpful error message and exits the CLI
// if the alias is detected as not being configured properly.
func MustBeConfigured(autoConfigure bool) error {
	if !IsConfigured() {
		return SetupShellWizard(autoConfigure)
	}
	return nil
}

// GetShellFromShellEnv returns the shell from the SHELL environment variable
func GetShellFromShellEnv(shellEnv string) (string, error) {
	if strings.Contains(shellEnv, "fish") {
		return "fish", nil

	} else if strings.Contains(shellEnv, "bash") {
		return "bash", nil

	} else if strings.Contains(shellEnv, "zsh") {
		return "zsh", nil

	} else {
		return "", fmt.Errorf("we couldn't detect your shell type (%s). Please follow the steps at https://granted.dev/shell-alias to assume roles with Granted", shellEnv)
	}
}

// GetShellAlias returns the alias config for a shell
func GetShellAlias(shell string) (Config, error) {
	var file string
	var err error
	alias := GetDefaultAlias()
	switch shell {
	case "fish":
		file, err = shells.GetFishConfigFile()
		alias = GetFishAlias()
	case "bash":
		file, err = shells.GetBashConfigFile()
	case "zsh":
		file, err = shells.GetZshConfigFile()
	default:
		err = &ErrShellNotSupported{Shell: shell}
	}
	if err != nil {
		return Config{}, err
	}
	return Config{File: file, Alias: alias}, nil
}

func SetupShellWizard(autoConfigure bool) error {
	// SHELL is set by the wrapper script
	shellEnv := os.Getenv("SHELL")
	shell, err := GetShellFromShellEnv(shellEnv)
	if err != nil {
		return err
	}
	cfg, err := GetShellAlias(shell)
	if err != nil {
		return err
	}
	// skip prompt if autoConfigure is set to true
	if !autoConfigure {
		ul := color.New(color.Underline).SprintFunc()
		fmt.Fprintf(color.Error, "ℹ️  To assume roles with Granted, we need to add an alias to your shell profile (%s).\n", ul("https://granted.dev/shell-alias"))
		withStdio := survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)
		in := &survey.Confirm{
			Message: fmt.Sprintf("Install %s alias at %s", shell, cfg.File),
			Default: true,
		}
		var confirm bool
		err = survey.AskOne(in, &confirm, withStdio)
		if err != nil {
			return err
		}

		if !confirm {
			return errors.New("cancelled alias installation")
		}

		fmt.Fprintln(color.Error, "")
	}

	err = Install(cfg)
	if err != nil {
		return err
	}
	fmt.Fprintf(color.Error, "Added the Granted alias to %s\n", cfg.File)
	alert := color.New(color.Bold, color.FgYellow).SprintFunc()
	fmt.Fprintf(color.Error, "\n%s\n", alert("Shell restart required to apply changes: please open a new terminal window and re-run your command."))
	os.Exit(0)
	return nil
}

type UninstallShellResults struct {
	ConfigFile string
}

// UninstallDefaultShellAlias tries to uninstall the Granted aliases from the
// user's default shell bindings
func UninstallDefaultShellAlias() error {
	// SHELL is set by the wrapper script
	shellEnv := os.Getenv("SHELL")
	shell, err := GetShellFromShellEnv(shellEnv)
	if err != nil {
		return fmt.Errorf("we couldn't detect your shell type (%s). You may need to manually removed Granted's aliases from your shell configuration", shellEnv)
	}
	cfg, err := GetShellAlias(shell)
	if err != nil {
		return err
	}
	err = Uninstall(cfg)
	if err != nil {
		return err
	}
	fmt.Fprintf(color.Error, "Removed the Granted alias from %s\n", cfg.File)
	return nil
}

// Install the Granted alias to a config file for a specified shell.
func Install(cfg Config) error {
	err := shells.AppendLine(cfg.File, cfg.Alias)
	var aee *shells.ErrLineAlreadyExists
	if errors.As(err, &aee) {
		return &ErrAlreadyInstalled{File: cfg.File}
	}
	return err
}

// Uninstall the Granted alias to a config file for a specified shell.
func Uninstall(cfg Config) error {
	err := shells.RemoveLine(cfg.File, cfg.Alias)
	var aee *shells.ErrLineAlreadyExists
	if errors.As(err, &aee) {
		return &ErrNotInstalled{File: cfg.File}
	}
	return err
}

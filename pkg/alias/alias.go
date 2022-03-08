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
	"io/ioutil"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
)

func init() {
	// By default the shell wrapper makes the color library
	// think that the terminal doesn't support colors.
	// We override this behaviour here so that we can print colored output.
	// Users can set NO_COLOR to true if they are working in a terminal without
	// color support and want to use Granted there.
	noColor := os.Getenv("NO_COLOR")
	color.NoColor = noColor == "true"
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
		_, err := SetupShellWizard(autoConfigure)
		if err != nil {
			return err
		}
	}
	return nil
}

// IsSupported returns true if Granted supports configuring aliases
// automatically for a user's shell
func IsSupported(shell string) bool {
	return shell == "fish" || shell == "bash" || shell == "zsh"
}

type SetupShellResults struct {
	ConfigFile string
}

func SetupShellWizard(autoConfigure bool) (*SetupShellResults, error) {
	// SHELL is set by the wrapper script
	shellEnv := os.Getenv("SHELL")
	fmt.Fprintf(os.Stderr, "shellEnv: %v\n", shellEnv)
	fmt.Fprintf(os.Stderr, "autoConfigure: %v\n", autoConfigure)
	var cfg Config
	var shell string
	var err error
	if strings.Contains(shellEnv, "fish") {
		shell = "fish"
		cfg, err = getFishConfig()
		if err != nil {
			return nil, err
		}
	} else if strings.Contains(shellEnv, "bash") {
		shell = "bash"
		cfg, err = getBashConfig()
		if err != nil {
			return nil, err
		}
	} else if strings.Contains(shellEnv, "zsh") {
		shell = "zsh"
		cfg, err = getZshConfig()
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("we couldn't detect your shell type (%s). Please follow the steps at https://granted.dev/shell-alias to assume roles with Granted", shellEnv)
	}

	// skip prompt if autoConfigure is set to true
	if !autoConfigure {
		ul := color.New(color.Underline).SprintFunc()

		fmt.Fprintf(os.Stderr, "ℹ️  To assume roles with Granted, we need to add an alias to your shell profile (%s).\n", ul("https://granted.dev/shell-alias"))

		label := fmt.Sprintf("Install %s alias at %s", shell, cfg.File)
		withStdio := survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)
		in := &survey.Confirm{
			Message: label,
			Default: true,
		}
		var confirm bool
		err = survey.AskOne(in, &confirm, withStdio)
		if err != nil {
			return nil, err
		}

		if !confirm {
			return nil, errors.New("cancelled alias installation")
		}

		fmt.Fprintln(os.Stderr, "")
	}

	err = install(cfg)
	if err != nil {
		return nil, err
	}

	alert := color.New(color.Bold, color.FgYellow).SprintFunc()
	fmt.Fprintf(os.Stderr, "\n%s\n", alert("Shell restart required to apply changes: please open a new terminal window and re-run your command."))
	os.Exit(0)

	r := SetupShellResults{
		ConfigFile: cfg.File,
	}
	return &r, nil
}

type UninstallShellResults struct {
	ConfigFile string
}

// UninstallDefaultShellAlias tries to uninstall the Granted aliases from the
// user's default shell bindings
func UninstallDefaultShellAlias() (*UninstallShellResults, error) {
	// SHELL is set by the wrapper script
	shellEnv := os.Getenv("SHELL")
	var cfg Config
	var err error
	if strings.Contains(shellEnv, "fish") {
		cfg, err = getFishConfig()
		if err != nil {
			return nil, err
		}
	} else if strings.Contains(shellEnv, "bash") {
		cfg, err = getBashConfig()
		if err != nil {
			return nil, err
		}
	} else if strings.Contains(shellEnv, "zsh") {
		cfg, err = getZshConfig()
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("We couldn't detect your shell type (%s). You may need to manually removed Granted's aliases from your shell configuration.", shellEnv)
	}

	err = uninstall(cfg)
	if err != nil {
		return nil, err
	}

	r := UninstallShellResults{
		ConfigFile: cfg.File,
	}
	return &r, nil
}

type ErrShellNotSupported struct {
	Shell string
}

func (e *ErrShellNotSupported) Error() string {
	return fmt.Sprintf("unsupported shell %s", e.Shell)
}

// Install the Granted alias to a config file for a specified shell.
func Install(shell string) error {
	var cfg Config
	var err error

	switch shell {
	case "fish":
		cfg, err = getFishConfig()
	case "bash":
		cfg, err = getBashConfig()
	case "zsh":
		cfg, err = getZshConfig()
	default:
		return &ErrShellNotSupported{Shell: shell}
	}
	if err != nil {
		return err
	}

	return install(cfg)
}

// Uninstall the Granted alias to a config file for a specified shell.
func Uninstall(shell string) error {
	var cfg Config
	var err error

	switch shell {
	case "fish":
		cfg, err = getFishConfig()
	case "bash":
		cfg, err = getBashConfig()
	case "zsh":
		cfg, err = getZshConfig()
	default:
		return &ErrShellNotSupported{Shell: shell}
	}
	if err != nil {
		return err
	}

	return uninstall(cfg)
}

type ErrAlreadyInstalled struct {
	File string
}

func (e *ErrAlreadyInstalled) Error() string {
	return fmt.Sprintf("the Granted alias has already been added to %s", e.File)
}

// install the Granted alias to a file.
// Returns ErrAlreadyInstalled if the alias
// already exists in the file.
func install(cfg Config) error {
	b, err := ioutil.ReadFile(cfg.File)
	if err != nil {
		return err
	}

	// return an error if the file already contains an
	// alias we've set up, to avoid Granted adding
	// an alias multiple times to a shell config file.
	if strings.Contains(string(b), cfg.Alias) {
		return &ErrAlreadyInstalled{File: cfg.File}
	}

	// open the file for writing
	out, err := os.OpenFile(cfg.File, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	// include newlines around the alias
	a := fmt.Sprintf("\n%s\n", cfg.Alias)
	_, err = out.WriteString(a)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Added the Granted alias to %s\n", cfg.File)
	return nil
}

type ErrNotInstalled struct {
	File string
}

func (e *ErrNotInstalled) Error() string {
	return fmt.Sprintf("the Granted alias hasn't been added to %s", e.File)
}

// uninstall the Granted alias to a file.
// Returns ErrNotInstalled if the alias
// doesn't exist in the file.
func uninstall(cfg Config) error {
	b, err := ioutil.ReadFile(cfg.File)
	if err != nil {
		return err
	}

	// the line number in the file where the alias was found
	var aliasLineIndex int
	var found bool

	var ignored []string

	lines := strings.Split(string(b), "\n")
	for i, line := range lines {
		removeLine := strings.Contains(line, cfg.Alias)

		// When removing the line containing the alias, if the line after the alias is empty we
		// remove that too. This prevents the length of the config file growing by 1 with blank lines
		// every time the Granted alias is installed and uninstalled. Really only useful as a nice
		// convenience for developing the Granted CLI, when we do a lot of installing/uninstalling the aliases.
		if found && i == aliasLineIndex+1 && line == "" {
			removeLine = true
		}

		if !removeLine {
			ignored = append(ignored, line)
		} else {
			// mark that we've found the alias in the file
			found = true
			aliasLineIndex = i
		}
	}

	if !found {
		// we didn't find the alias in the file, so return an error in order to let the user know that it doesn't exist there.
		return &ErrNotInstalled{File: cfg.File}
	}

	output := strings.Join(ignored, "\n")

	err = ioutil.WriteFile(cfg.File, []byte(output), 0644)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Removed the Granted alias from %s\n", cfg.File)
	return nil
}

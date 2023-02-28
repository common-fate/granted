package launcher

import "github.com/common-fate/granted/pkg/browser"

// Open calls the 'open' command to open a URL.
// This is the same command as when you run 'open -a Safari https://commonfate.io'
// in your own terminal.
type Safari struct{}

func (l Safari) LaunchCommand(url string, profile string) []string {
	cmd := browser.OpenCommand()
	return []string{cmd, "-a", "Safari", url}
}

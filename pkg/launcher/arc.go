package launcher

import "github.com/common-fate/granted/pkg/browser"

type Arc struct {
}

func (l Arc) LaunchCommand(url string, profile string) []string {
	cmd := browser.OpenCommand()
	return []string{cmd, "-a", "Arc", url}
}

func (l Arc) UseForkProcess() bool { return false }

package launcher

import "fmt"

type Console struct {
	ExecutablePath string
}

func (l Console) LaunchCommand(url string, profile string) []string {
	return []string{l.ExecutablePath, fmt.Sprintf("--profile='%s' --url='%s'", profile, url)}
}

func (l Console) UseForkProcess() bool { return false }

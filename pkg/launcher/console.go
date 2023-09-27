package launcher

import "fmt"

type Console struct {
	ExecutablePath string
}

func (l Console) LaunchCommand(url string, profile string) []string {
	return []string{fmt.Sprintf("%s --profile='%s' --url='%s'", l.ExecutablePath, profile, url)}
}

func (l Console) UseForkProcess() bool { return false }

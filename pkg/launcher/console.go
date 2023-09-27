package launcher

type Console struct {
	ExecutablePath string
}

func (l Console) LaunchCommand(url string, profile string) []string {
	return []string{l.ExecutablePath,
		"--profile=" + profile,
		"--url=" + url,
	}
}

func (l Console) UseForkProcess() bool { return false }

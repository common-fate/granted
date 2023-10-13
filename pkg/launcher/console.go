package launcher

type CommonFate struct {
	ExecutablePath string
}

func (l CommonFate) LaunchCommand(url string, profile string) []string {
	return []string{l.ExecutablePath,
		"--profile=" + profile,
		"--url=" + url,
	}
}

func (l CommonFate) UseForkProcess() bool { return false }

package launcher

type FirefoxNightly struct {
	ExecutablePath string
}

func (l FirefoxNightly) LaunchCommand(url string, profile string) []string {
	return []string{
		l.ExecutablePath,
		"--new-tab",
		url,
	}
}

func (l FirefoxNightly) UseForkProcess() bool { return true }

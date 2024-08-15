package launcher

type FirefoxNightly struct {
	ExecutablePath string
}

func (l FirefoxNightly) LaunchCommand(url string, profile string) ([]string, error) {
	return []string{
		l.ExecutablePath,
		"--new-tab",
		url,
	}, nil
}

func (l FirefoxNightly) UseForkProcess() bool { return true }

package launcher

type FirefoxDevEdition struct {
	ExecutablePath string
}

func (l FirefoxDevEdition) LaunchCommand(url string, profile string) ([]string, error) {
	return []string{
		l.ExecutablePath,
		"--new-tab",
		url,
	}, nil
}

func (l FirefoxDevEdition) UseForkProcess() bool { return true }

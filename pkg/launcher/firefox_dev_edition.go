package launcher

type FirefoxDevEdition struct {
	ExecutablePath string
}

func (l FirefoxDevEdition) LaunchCommand(url string, profile string) []string {
	return []string{
		l.ExecutablePath,
		"--new-tab",
		url,
	}
}

func (l FirefoxDevEdition) UseForkProcess() bool { return true }

package launcher

type Firefox struct {
	// ExecutablePath is the path to the Firefox binary on the system.
	ExecutablePath string
}

func (l Firefox) LaunchCommand(url string, profile string) ([]string, error) {
	return []string{
		l.ExecutablePath,
		"--new-tab",
		url,
	}, nil
}

func (l Firefox) UseForkProcess() bool { return true }

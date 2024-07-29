package launcher

type Flatpak struct {
	// ExecutablePath is the path to the Flatpak binary on the system.
	ExecutablePath string

	// FlatpakID is the ID of the Flatpak to run.
	FlatpakID string

	// BrowserType is the type of browser to expect (e.g. FIREFOX).
	BrowserType string
}

func (l Flatpak) LaunchCommand(url string, profile string) []string {
	switch l.BrowserType {
	case "FIREFOX":
		return []string{
			l.ExecutablePath,
			"run",
			l.FlatpakID,
			"--new-tab",
			url,
		}
	default:
		return []string{}
	}
}

func (l Flatpak) UseForkProcess() bool { return true }

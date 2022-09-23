package launcher

// Open calls the 'open' command to open a URL.
// This is the same command as when you run 'open https://commonfate.io'
// in your own terminal.
type Open struct{}

func (l Open) LaunchCommand(url string, profile string) []string {
	return []string{"open", url}
}

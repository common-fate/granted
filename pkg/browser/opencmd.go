package browser

import (
	"os/exec"
	"runtime"
)

// OpenCommand returns the terminal command to open a browser.
// This is system dependent - for MacOS we use 'open',
// whereas for Linux we use 'xdg-open', 'x-www-browser', or 'www-browser'.
func OpenCommand() string {
	if runtime.GOOS == "linux" {
		cmds := []string{"xdg-open", "x-www-browser", "www-browser"}
		for _, c := range cmds {
			if _, err := exec.LookPath(c); err == nil {
				return c
			}
		}
	}

	return "open"
}

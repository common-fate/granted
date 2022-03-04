package browsers

import (
	"os"
	"os/exec"

	"github.com/common-fate/granted/pkg/debug"
)

func HandleLinuxBrowserSearch() (string, error) {
	out, err := exec.Command("xdg-settings", "get", "default-web-browser").Output()

	if err != nil {
		debug.Fprintf(debug.VerbosityDebug, os.Stderr, err.Error())
	}

	return string(out), nil
}

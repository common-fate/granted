package browser

import (
	"os/exec"

	"github.com/common-fate/granted/pkg/debug"
	"github.com/fatih/color"
)

func HandleLinuxBrowserSearch() (string, error) {
	out, err := exec.Command("xdg-settings", "get", "default-web-browser").Output()

	if err != nil {
		debug.Fprintf(debug.VerbosityDebug, color.Error, err.Error())
	}

	return string(out), nil
}

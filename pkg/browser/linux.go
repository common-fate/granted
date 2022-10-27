package browser

import (
	"os/exec"

	"github.com/common-fate/clio"
)

func HandleLinuxBrowserSearch() (string, error) {
	out, err := exec.Command("xdg-settings", "get", "default-web-browser").Output()

	if err != nil {
		clio.Debug(err.Error())
	}

	return string(out), nil
}

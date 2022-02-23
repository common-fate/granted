package browsers

import "os/exec"

func HandleLinuxBrowserSearch() (string, error) {
	out, err := exec.Command("xdg-settings", "get", "default-web-browser").Output()

	if err != nil {
		return "", err
	}

	return string(out), nil
}

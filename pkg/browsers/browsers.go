package browsers

import (
	"errors"
	"os/exec"
	"runtime"
)

const (
	ChromeKey   string = "CHROME"
	FirefoxKey  string = "FIREFOX"
	EdgeKey     string = "EDGE"
	BraveKey    string = "BRAVE"
	DefaultKey  string = "DEFAULT"
	ChromiumKey string = "CHROMIUM"
)

const ChromePathMac = "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
const ChromePathLinux = `/usr/bin/google-chrome`
const ChromePathWindows = `\Program Files\Google\Chrome\Application\chrome.exe`
const BravePathMac = "/Applications/Brave Browser.app/Contents/MacOS/Brave Browser"
const BravePathLinux = `/usr/bin/brave-browser`
const BravePathWindows = `\Program Files\BraveSoftware\Brave-Browser\Application\brave.exe`
const EdgePathMac = "/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge"
const EdgePathLinux = `/usr/bin/edge`
const EdgePathWindows = `\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`
const FirefoxPathMac = "/Applications/Firefox.app/Contents/MacOS/firefox"
const FirefoxPathLinux = `/usr/bin/firefox`
const FirefoxPathWindows = `\Program Files\Mozilla Firefox\firefox.exe`
const ChromiumPathMac = "/Applications/Chromium.app/Contents/MacOS/Chromium"
const ChromiumPathLinux = `/usr/bin/chromium`
const ChromiumPathWindows = `\Program Files\Chromium\chromium.exe`

func ChromePath() (string, error) {
	//check linuxpath for binary install
	path, err := exec.LookPath("chrome")
	if err == nil {
		return path, err
	}
	switch runtime.GOOS {
	case "windows":
		return ChromePathWindows, nil
	case "darwin":
		return ChromePathMac, nil
	case "linux":
		return ChromePathLinux, nil
	default:
		return "", errors.New("os not supported")
	}
}
func BravePath() (string, error) {
	//check linuxpath for binary install
	path, err := exec.LookPath("brave")
	if err == nil {
		return path, err
	}
	switch runtime.GOOS {
	case "windows":
		return BravePathWindows, nil
	case "darwin":
		return BravePathMac, nil
	case "linux":

		return BravePathLinux, nil
	default:
		return "", errors.New("os not supported")
	}
}
func EdgePath() (string, error) {
	//check linuxpath for binary install
	path, err := exec.LookPath("edge")
	if err == nil {
		return path, err
	}
	switch runtime.GOOS {
	case "windows":
		return EdgePathWindows, nil
	case "darwin":
		return EdgePathMac, nil
	case "linux":

		return EdgePathLinux, nil
	default:
		return "", errors.New("os not supported")
	}
}
func FirefoxPath() (string, error) {
	//check linuxpath for binary install
	path, err := exec.LookPath("firefox")
	if err == nil {
		return path, err
	}
	switch runtime.GOOS {
	case "windows":
		return FirefoxPathWindows, nil
	case "darwin":
		return FirefoxPathMac, nil
	case "linux":

		return FirefoxPathLinux, nil
	default:
		return "", errors.New("os not supported")
	}
}

func ChromiumPath() (string, error) {
	//check linuxpath for binary install
	path, err := exec.LookPath("chromium")
	if err == nil {
		return path, err
	}
	switch runtime.GOOS {
	case "windows":
		return ChromiumPathWindows, nil
	case "darwin":
		return ChromiumPathMac, nil
	case "linux":

		return ChromiumPathLinux, nil
	default:
		return "", errors.New("os not supported")
	}
}

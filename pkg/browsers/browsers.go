package browsers

import (
	"errors"
	"runtime"
)

const (
	ChromeKey  string = "CHROME"
	FirefoxKey string = "FIREFOX"
	EdgeKey    string = "EDGE"
	BraveKey   string = "BRAVE"
	DefaultKey string = "DEFAULT"
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

func ChromePath() (string, error) {
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

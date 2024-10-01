//go:build !windows
// +build !windows

package chromemsg

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/common-fate/clio"
)

// Configure writes native messaging host configuration to various well-known folders,
// including Google Chrome, Arc, Microsoft Edge, and Vivaldi.
//
// See: https://developer.chrome.com/docs/extensions/develop/concepts/native-messaging#native-messaging-host
//
// The resulting file looks like this:
//
//	  {
//		 "name": "io.commonfate.granted",
//		 "description": "Granted BrowserSupport",
//		 "path": "/usr/local/bin/granted",
//		 "type": "stdio",
//		 "allowed_origins": [
//		   "chrome-extension://fcipjekpmlpmiikgdecbjbcpmenmceoh/"
//		 ]
//	  }
func ConfigureHost() error {
	switch runtime.GOOS {
	case "darwin":
		return configureMacOS()
	case "linux":
		return configureLinux()
	}

	return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
}

func configureMacOS() error {
	appSupportFolder := filepath.Join(os.Getenv("HOME"), "Library", "Application Support")
	browsers := []string{
		"Arc/User Data",
		"Google/Chrome Beta",
		"Google/Chrome Canary",
		"Google/Chrome Dev",
		"Chromium",
		"Microsoft Edge",
		"Microsoft Edge Beta",
		"Microsoft Edge Canary",
		"Microsoft Edge Dev",
		"Vivaldi",
	}

	for _, browser := range browsers {
		browserPath := filepath.Join(appSupportFolder, browser, "NativeMessagingHosts")
		if _, err := os.Stat(browserPath); os.IsNotExist(err) {
			continue
		}

		manifestPath := filepath.Join(browserPath, "io.commonfate.granted.json")
		if err := writeManifest(manifestPath); err != nil {
			return err
		}

		clio.Debugf("wrote native messaging manifest: %s", manifestPath)
	}
	return nil
}

func configureLinux() error {
	browsers := []string{
		filepath.Join(os.Getenv("HOME"), ".config/google-chrome/NativeMessagingHosts"),
		filepath.Join(os.Getenv("HOME"), ".config/chromium/NativeMessagingHosts"),
	}

	for _, browserPath := range browsers {
		if _, err := os.Stat(browserPath); err == nil {
			manifestPath := filepath.Join(browserPath, "io.commonfate.granted.json")
			if err := writeManifest(manifestPath); err != nil {
				return err
			}
		}
	}
	return nil
}

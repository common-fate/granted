//go:build windows
// +build windows

package chromemsg

import (
	"path/filepath"

	"github.com/common-fate/granted/pkg/config"
	"golang.org/x/sys/windows/registry"
)

func ConfigureHost() error {
	return configureWindows()
}

func configureWindows() error {
	// Registry paths
	paths := []string{
		"Software\\Google\\Chrome\\NativeMessagingHosts",
		"Software\\Chromium\\NativeMessagingHosts",
	}

	grantedConfigFolder, err := config.GrantedConfigFolder()
	if err != nil {
		return err
	}

	manifestPath := filepath.Join(grantedConfigFolder, "native-messaging-host-manifest.json")
	err = writeManifest(manifestPath)
	if err != nil {
		return err
	}

	for _, regPath := range paths {
		key, err := registry.OpenKey(registry.CURRENT_USER, regPath, registry.QUERY_VALUE|registry.SET_VALUE)
		if err != nil {
			continue
		}
		defer key.Close()

		gkey, _, err := registry.CreateKey(key, "io.commonfate.granted", registry.QUERY_VALUE|registry.SET_VALUE)
		if err != nil {
			continue
		}
		defer gkey.Close()

		if err = gkey.SetStringValue("", manifestPath); err != nil {
			return err
		}
	}
	return nil
}

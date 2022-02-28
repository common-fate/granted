//go:build windows

package browsers

import (
	"golang.org/x/sys/windows/registry"
)

func HandleWindowsBrowserSearch() (string, error) {
	// Lookup https handler in registry
	k, err := registry.OpenKey(registry.CURRENT_USER, `SOFTWARE\\Microsoft\\Windows\\Shell\\Associations\\UrlAssociations\\https\\UserChoice`, registry.QUERY_VALUE)
	if err != nil {
		return "", err
	}
	kv, _, err := k.GetStringValue("ProgId")
	if err != nil {
		return "", err
	}
	return kv, nil
}

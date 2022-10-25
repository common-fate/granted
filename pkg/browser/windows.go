//go:build windows

package browser

import (
	"github.com/common-fate/clio"
	"golang.org/x/sys/windows/registry"
)

func HandleWindowsBrowserSearch() (string, error) {
	// Lookup https handler in registry
	k, err := registry.OpenKey(registry.CURRENT_USER, `SOFTWARE\\Microsoft\\Windows\\Shell\\Associations\\UrlAssociations\\https\\UserChoice`, registry.QUERY_VALUE)
	if err != nil {
		clio.Debug(err.Error())
	}
	kv, _, err := k.GetStringValue("ProgId")
	if err != nil {
		clio.Debug(err.Error())
	}
	return kv, nil
}

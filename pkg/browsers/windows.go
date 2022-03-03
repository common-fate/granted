//go:build windows

package browsers

import (
	"os"

	"github.com/common-fate/granted/pkg/debug"
	"golang.org/x/sys/windows/registry"
)

func HandleWindowsBrowserSearch() (string, error) {
	// Lookup https handler in registry
	k, err := registry.OpenKey(registry.CURRENT_USER, `SOFTWARE\\Microsoft\\Windows\\Shell\\Associations\\UrlAssociations\\https\\UserChoice`, registry.QUERY_VALUE)
	if err != nil {
		debug.Fprintf(debug.VerbosityDebug, os.Stderr, err.Error())
	}
	kv, _, err := k.GetStringValue("ProgId")
	if err != nil {
		debug.Fprintf(debug.VerbosityDebug, os.Stderr, err.Error())
	}
	return kv, nil
}

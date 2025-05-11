// Package awsmerge contains logic to merge multiple AWS config files
// together. In Granted this is used to power the Profile Registry feature.
package awsmerge

import (
	"gopkg.in/ini.v1"
)

// RemoveRegistry removes a profile registry section from an AWS config file.
//
// It removes a range around the generated profiles as follows:
//
//	[granted_registry_start test]
//	// ... <generated profiles>
//	[granted_registry_end test]
func RemoveRegistry(src *ini.File, name string) {
	// replace any existing generated profiles which match the name of the registry
	existing := getGrantedGeneratedSections(src, name)
	for _, p := range existing {
		src.DeleteSection(p.Name())
	}
}

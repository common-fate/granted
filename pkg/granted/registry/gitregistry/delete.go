package gitregistry

import (
	"os"
)

// Delete the local clone of the registry repo.
func (r Registry) Delete() error {
	err := os.RemoveAll(r.clonedTo)
	if err != nil {
		return err
	}

	return nil
}

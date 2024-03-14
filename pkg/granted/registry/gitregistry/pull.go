package gitregistry

import (
	"os"

	"github.com/common-fate/granted/pkg/git"
)

// pull ensures the remote git repo is cloned and the latest changes are pulled.
func (r Registry) pull() error {
	if _, err := os.Stat(r.clonedTo); err != nil {
		// folder doesn't exist yet, so clone the repo and return early.
		return git.Clone(r.opts.URL, r.clonedTo)
	}

	// if we get here, the folder exists, so pull any changes.
	err := git.Pull(r.clonedTo, false)
	if err != nil {
		return err
	}

	return nil
}

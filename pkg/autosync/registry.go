package autosync

import (
	"errors"
	"fmt"
	"time"

	"github.com/common-fate/clio"
	"github.com/common-fate/granted/pkg/granted/registry"
)

type RegistrySyncError struct {
	err error
}

func (e *RegistrySyncError) Error() string {
	return fmt.Sprintf("error syncing profile registry with err: %s", e.err.Error())
}

func runSync(rc RegistrySyncConfig) {
	defer waitgroup.Done()
	if err := registry.SyncProfileRegistries(); err != nil {

		checks.mu.Lock()
		checks.errs = append(checks.errs, &RegistrySyncError{err: err})
		checks.mu.Unlock()

		return
	}

	rc.LastCheckForSync = time.Now().Weekday()
	if err := rc.Save(); err != nil {
		clio.Debug("unable to save to registry sync config")

		checks.mu.Lock()
		checks.errs = append(checks.errs, &RegistrySyncError{err: errors.New(err.Error())})
		checks.mu.Unlock()
		return
	}

	checks.mu.Lock()
	checks.msgs = append(checks.msgs, fmt.Sprintf("successfully synced for the day %s", time.Now()))
	checks.mu.Unlock()
}

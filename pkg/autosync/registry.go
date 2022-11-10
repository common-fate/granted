package autosync

import (
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

func runSync(rc RegistrySyncConfig) error {
	clio.Info("Syncing Profile Registries")
	shouldSilentLog := true
	err := registry.SyncProfileRegistries(shouldSilentLog)
	if err != nil {
		return &RegistrySyncError{err: err}
	}
	rc.LastCheckForSync = time.Now().Weekday()
	err = rc.Save()
	if err != nil {
		clio.Debug("unable to save to registry sync config")
		return &RegistrySyncError{err: err}
	}
	clio.Success("Completed syncing Profile Registries")
	return nil
}

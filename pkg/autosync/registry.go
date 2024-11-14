package autosync

import (
	"context"
	"fmt"
	"time"

	"github.com/common-fate/clio"
	"github.com/common-fate/granted/pkg/granted/registry"
)

// This error is not needed, as it is not being used anywhere and
// does not offer us an specific behaviour on the event of the error
// occurring, so we will just eliminate it
//
// type RegistrySyncError struct {
// 	err error
// }

// func (e *RegistrySyncError) Error() string {
// 	return fmt.Sprintf("error syncing profile registry with err: %s", e.err.Error())
// }
// type RegistrySyncError struct {
// 	err error
// }

// func (e *RegistrySyncError) Error() string {
// 	return fmt.Sprintf("error syncing profile registry with err: %s", e.err.Error())
// }

func runSync(ctx context.Context, rc RegistrySyncConfig, interactive bool) error {
	clio.Info("Syncing Profile Registries")
	err := registry.SyncProfileRegistries(ctx, interactive)
	if err != nil {
		return err
	}
	rc.LastCheckForSync = time.Now().Weekday()
	err = rc.Save()
	if err != nil {
		return fmt.Errorf("saving registry sync config: %w", err)
	}
	clio.Success("Completed syncing Profile Registries")
	return nil
}

package autosync

import (
	"context"
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

package autosync

import (
	"time"

	"github.com/common-fate/clio"
	grantedConfig "github.com/common-fate/granted/pkg/config"
)

func Run() {
	// check if registry has been configured or not.
	// should skip registry sync if no profile registry.
	gConf, err := grantedConfig.Load()
	if err != nil {
		clio.Debugf("unable to load granted config file with err %s", err.Error())
		return
	}

	if len(gConf.ProfileRegistryURLS) < 1 {
		clio.Debug("profile registry not configured. Skipping auto sync.")
		return
	}

	// load and check if sync has been run for the day. If true then skip.
	rc, ok := loadRegistryConfig()
	clio.Debug("checking if autosync has been run for the day")
	if ok && time.Now().Weekday() == rc.LastCheckForSync {
		clio.Debug("skipping profile registry sync until tomorrow=%s", rc.Path())
		return
	}

	err = runSync(rc)
	if err != nil {
		clio.Debugw("failed to sync profile registries", "error", err)
		clio.Warn("Failed to sync Profile Registries")
	}
}

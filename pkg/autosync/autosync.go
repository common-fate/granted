package autosync

import (
	"sync"
	"time"

	"github.com/common-fate/clio"
	grantedConfig "github.com/common-fate/granted/pkg/config"
)

var waitgroup sync.WaitGroup

var checks struct {
	mu   sync.Mutex
	msgs []string
	errs []error
}

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

	waitgroup.Add(1)
	go runSync(rc)
}

// Print the output of the sync for that day.
func Print() {
	waitgroup.Wait()
	for _, msg := range checks.msgs {
		if msg != "" {
			clio.Debug(msg)
		}
	}

	for _, err := range checks.errs {
		if err != nil {
			clio.Warn(err)
		}
	}
}

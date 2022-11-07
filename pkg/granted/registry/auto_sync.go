package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"sync"
	"time"

	"github.com/common-fate/clio"
)

// waitgroup is used to ensure that Check() has finished
var waitgroup sync.WaitGroup

var checks struct {
	mu   sync.Mutex
	msgs []string
}

const (
	FILENAME = "registry-sync"
)

type registrySyncConfig struct {
	dir              string
	LastCheckForSync time.Weekday `json:"lastCheckForSync"`
}

func (rc registrySyncConfig) Path() string {
	return path.Join(rc.dir, FILENAME)
}

func (rc registrySyncConfig) Save() error {
	if rc.dir == "" {
		return errors.New("version config dir was not specified")
	}

	err := os.MkdirAll(rc.dir, os.ModePerm)
	if err != nil {
		return err
	}

	data, err := json.Marshal(rc)
	if err != nil {
		return err
	}

	os.WriteFile(rc.Path(), data, 0700)
	return nil
}

func loadRegistryConfig() (rc registrySyncConfig, ok bool) {
	cd, err := os.UserConfigDir()
	if err != nil {
		clio.Debug("error loading user config dir: %s", err.Error())
		return
	}

	rc.dir = path.Join(cd, "commonfate")
	err = os.MkdirAll(rc.dir, os.ModePerm)
	if err != nil {
		clio.Debug("error creating commonfate config dir: %s", err.Error())
		return
	}

	rcfile := path.Join(rc.dir, FILENAME)
	if _, err := os.Stat(rcfile); errors.Is(err, os.ErrNotExist) {
		clio.Debug("version config file does not exist: %s", rcfile)
		return
	}

	data, err := os.ReadFile(rcfile)
	if err != nil {
		clio.Debug("error reading version config: %s", err.Error())
		return
	}
	err = json.Unmarshal(data, &rc)
	if err != nil {
		clio.Debug("error unmarshalling version config: %s", err.Error())
		return
	}
	ok = true
	return
}

func AutoSync() error {
	// load and check if sync has been run for today.
	rc, ok := loadRegistryConfig()
	if ok && time.Now().Weekday() == rc.LastCheckForSync {
		clio.Debug("skipping profile registry sync until tomorrow=%s", rc.Path())

		return nil
	}

	checks.mu.Lock()
	defer checks.mu.Unlock()
	checks.msgs = nil

	waitgroup.Add(1)
	go runSync(rc)

	return nil
}

// Print the status of sync for the day.
func Print() {
	waitgroup.Wait()
	for _, msg := range checks.msgs {
		if msg != "" {
			clio.Debug(msg)
		}
	}
}

func runSync(rc registrySyncConfig) error {
	defer waitgroup.Done()
	if err := syncProfileRegistries(); err != nil {
		return err
	}

	rc.LastCheckForSync = time.Now().Weekday()
	if err := rc.Save(); err != nil {
		return err
	}

	checks.mu.Lock()
	defer checks.mu.Unlock()
	checks.msgs = append(checks.msgs, fmt.Sprintf("successfully synced for the day %s", time.Now()))

	return nil
}

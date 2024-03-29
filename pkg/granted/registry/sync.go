package registry

import (
	"fmt"
	"io"
	"os"
	"path"

	"github.com/common-fate/clio"
	grantedConfig "github.com/common-fate/granted/pkg/config"
	"github.com/urfave/cli/v2"
	"gopkg.in/ini.v1"
)

const (
	SyncTempDirPrefix = "granted-registry-sync"
)

var SyncCommand = cli.Command{
	Name:        "sync",
	Usage:       "Pull the latest change from remote origin and sync aws profiles in aws config files",
	Description: "Pull the latest change from remote origin and sync aws profiles in aws config files",
	Action: func(c *cli.Context) error {
		if err := SyncProfileRegistries(false, true, false); err != nil {
			return err
		}

		return nil
	},
}

type syncOpts struct {
	isFirstSection                 bool
	promptUserIfProfileDuplication bool
	shouldSilentLog                bool
	shouldFailForRequiredKeys      bool
}

// Wrapper around sync func. Check if profile registry is configured, pull the latest changes and call sync func.
// promptUserIfProfileDuplication if true will automatically prefix the duplicate profiles and won't prompt users
// this is useful when new registry with higher priority is added and there is duplication with lower priority registry.
func SyncProfileRegistries(shouldSilentLog bool, promptUserIfProfileDuplication bool, shouldFailForRequiredKeys bool) error {
	registries, err := GetProfileRegistries()
	if err != nil {
		return err
	}

	if len(registries) == 0 {
		clio.Warn("granted registry not configured. Try adding a git repository with 'granted registry add <https://github.com/your-org/your-registry.git>'")
	}

	awsConfigPath, err := getDefaultAWSConfigLocation()
	if err != nil {
		return err
	}

	tmpDir, err := os.MkdirTemp("", SyncTempDirPrefix)
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	if err := os.Chmod(tmpDir, 0755); err != nil {
		return err
	}

	tmpConfigPath := path.Join(tmpDir, "aws-config")

	if err := createFileIfNotExists(awsConfigPath); err != nil {
		return err
	}
	if err := copyFile(awsConfigPath, tmpConfigPath); err != nil {

		return fmt.Errorf("failed to copy aws config to tempfile for update")
	}

	configFile, err := loadAWSConfigFileFromPath(tmpConfigPath)
	if err != nil {
		return err
	}

	// if the config file contains granted generated content then remove it
	if err := removeAutogeneratedProfiles(configFile, tmpConfigPath); err != nil {
		return err
	}

	for index, r := range registries {
		isFirstSection := false
		if index == 0 {
			isFirstSection = true
		}

		err = runSync(&r, configFile, tmpConfigPath, syncOpts{
			isFirstSection:                 isFirstSection,
			shouldSilentLog:                shouldSilentLog,
			promptUserIfProfileDuplication: promptUserIfProfileDuplication,
			shouldFailForRequiredKeys:      shouldFailForRequiredKeys,
		})

		if err != nil {
			se, ok := err.(*SyncError)
			if ok {
				clio.Errorf("Sync failed for registry %s: %s", r.Config.Name, se.Error())

				if r.Config.WriteOnSyncFailure {
					clio.Warnf("%s is configured to write on sync failure; continuing.", r.Config.Name)
					continue
				}
			}
			return err
		}
	}

	// Run only if all syncs have succeeded
	clio.Debugf("sync successful; moving %s to %s", tmpConfigPath, awsConfigPath)
	if err := os.Rename(tmpConfigPath, awsConfigPath); err != nil {
		return err
	}

	return nil
}

// runSync will return custom error when there is error for specific registry so that
// other registries can still be synced.
func runSync(r *Registry, configFile *ini.File, configFilePath string, opts syncOpts) error {
	repoDirPath, err := getRegistryLocation(r.Config)
	if err != nil {
		return err
	}

	// If the local repo has been deleted, then attempt to clone it again
	_, err = os.Stat(repoDirPath)
	if os.IsNotExist(err) {
		err = gitClone(r.Config.URL, repoDirPath)
		if err != nil {
			return &SyncError{
				RegistryName: r.Config.Name,
				Err:          err,
			}
		}
	} else {
		err = gitPull(repoDirPath, opts.shouldSilentLog)
		if err != nil {
			return &SyncError{
				RegistryName: r.Config.Name,
				Err:          err,
			}
		}
	}

	err = r.Parse()
	if err != nil {
		return &SyncError{
			RegistryName: r.Config.Name,
			Err:          err,
		}
	}

	err = r.PromptRequiredKeys([]string{}, opts.shouldFailForRequiredKeys)
	if err != nil {
		return err
	}

	if err := Sync(r, configFile, opts); err != nil {
		return err
	}

	err = configFile.SaveTo(configFilePath)
	if err != nil {
		return err
	}

	return nil
}

// when there is new duplication when running sync command
// and if user chooses to duplicate then currently the config is not saved to gconfig.

// Sync function will load all the configs provided in the clonedFile.
// and generate a new section in the ~/.aws/profile file.
func Sync(r *Registry, awsConfigFile *ini.File, opts syncOpts) error {
	clio.Debugf("syncing %s \n", r.Config.Name)

	clonedFile, err := loadClonedConfigs(*r)
	if err != nil {
		return err
	}

	gconf, err := grantedConfig.Load()
	if err != nil {
		return err
	}

	// return custom error that should be caught and skipped.
	err = generateNewRegistrySection(r, awsConfigFile, clonedFile, gconf, opts)
	if err != nil {
		return &SyncError{
			Err:          err,
			RegistryName: r.Config.Name,
		}
	}

	clio.Successf("Successfully synced registry %s", r.Config.Name)

	return nil
}

type SyncError struct {
	Err          error
	RegistryName string
}

func (m *SyncError) Error() string {
	return fmt.Sprintf("Failed to sync for registry %s with error: %s", m.RegistryName, m.Err.Error())
}

func createFileIfNotExists(path string) error {
	_, statErr := os.Stat(path)
	if statErr == nil {
		return nil
	}

	if !os.IsNotExist(statErr) {
		return fmt.Errorf(`failed to check if file "%s" exists: %w`, path, statErr)
	}

	createdFile, createErr := os.Create(path)
	if createErr != nil {
		return fmt.Errorf(`failed to create file "%s": %w`, path, createErr)
	}
	defer createdFile.Close()

	return nil
}

func copyFile(from, to string) error {
	copyFrom, err := os.Open(from)
	if err != nil {
		return err
	}
	defer copyFrom.Close()
	copyTo, err := os.Create(to)
	if err != nil {
		return err
	}
	defer copyTo.Close()

	_, err = io.Copy(copyTo, copyFrom)
	return err
}

package registry

import (
	"fmt"
	"os"
	"path"

	"github.com/common-fate/clio"
	grantedConfig "github.com/common-fate/granted/pkg/config"

	"github.com/urfave/cli/v2"
)

const (
	// permission for user to read/write/execute.
	USER_READ_WRITE_PERM = 0700
)

var AddCommand = cli.Command{
	Name:        "add",
	Description: "Add a Profile Registry that you want to sync with aws config file",
	Usage:       "Provide git repository you want to sync with aws config file",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "name", Required: true, Usage: "name is used to uniquely identify profile registries", Aliases: []string{"n"}},
		&cli.StringFlag{Name: "url", Required: true, Usage: "git url for the remote repository", Aliases: []string{"u"}},
		&cli.StringFlag{Name: "path", Usage: "provide path if only the subfolder needs to be synced", Aliases: []string{"p"}},
		&cli.StringFlag{Name: "filename", Aliases: []string{"f"}, Usage: "provide filename if yml file is not granted.yml", DefaultText: "granted.yml"},
		&cli.IntFlag{Name: "priority", Usage: "profile registry will be sorted by priority descending", Value: 0},
		&cli.StringFlag{Name: "ref", Hidden: true},
		&cli.BoolFlag{Name: "prefix-all-profiles", Aliases: []string{"pap"}, Usage: "provide this flag if you want to append registry name to all profiles"},
		&cli.BoolFlag{Name: "prefix-duplicate-profiles", Aliases: []string{"pdp"}, Usage: "provide this flag if you want to append registry name to duplicate profiles"},
		&cli.BoolFlag{Name: "write-on-sync-failure", Aliases: []string{"wosf"}, Usage: "always overwrite AWS config, even if sync fails"},
		&cli.StringSliceFlag{Name: "required-key", Aliases: []string{"r", "requiredKey"}, Usage: "used to bypass the prompt or override user specific values"}},
	ArgsUsage: "--name <registry_name> --url <repository_url>",
	Action: func(c *cli.Context) error {
		gConf, err := grantedConfig.Load()
		if err != nil {
			return err
		}

		name := c.String("name")
		gitURL := c.String("url")
		pathFlag := c.String("path")
		configFileName := c.String("filename")
		ref := c.String("ref")
		prefixAllProfiles := c.Bool("prefix-all-profiles")
		prefixDuplicateProfiles := c.Bool("prefix-duplicate-profiles")
		writeOnSyncFailure := c.Bool("write-on-sync-failure")
		requiredKey := c.StringSlice("required-key")
		priority := c.Int("priority")

		for _, r := range gConf.ProfileRegistry.Registries {
			if r.Name == name {
				clio.Errorf("profile registry with name '%s' already exists. Name is required to be unique. Try adding with different name.\n", name)

				return nil
			}
		}

		registry := NewProfileRegistry(registryOptions{
			name:                    name,
			path:                    pathFlag,
			configFileName:          configFileName,
			url:                     gitURL,
			ref:                     ref,
			priority:                priority,
			prefixAllProfiles:       prefixAllProfiles,
			prefixDuplicateProfiles: prefixDuplicateProfiles,
			writeOnSyncFailure:      writeOnSyncFailure,
		})

		repoDirPath, err := getRegistryLocation(registry.Config)
		if err != nil {
			return err
		}

		if _, err = os.Stat(repoDirPath); err != nil {
			err = gitClone(gitURL, repoDirPath)
			if err != nil {
				return err
			}

			// //if a specific ref is passed we will checkout that ref
			// if ref != "" {
			// 	fmt.Println("attempting to checkout branch" + ref)

			// 	err = checkoutRef(ref, repoDirPath)
			// 	if err != nil {
			// 		return err

			// 	}
			// }

		} else {
			err = gitPull(repoDirPath, false)
			if err != nil {
				return err
			}
		}

		err = registry.Parse()
		if err != nil {
			return err
		}

		err = registry.PromptRequiredKeys(requiredKey, false)
		if err != nil {
			return err
		}

		awsConfigPath, err := getDefaultAWSConfigLocation()
		if err != nil {
			return err
		}

		if _, err := os.Stat(awsConfigPath); os.IsNotExist(err) {
			clio.Debugf("%s file does not exist. Creating an empty file\n", awsConfigPath)

			// create all parent directory if necessary.
			err := os.MkdirAll(path.Dir(awsConfigPath), USER_READ_WRITE_PERM)
			if err != nil {
				return err
			}

			_, err = os.Create(awsConfigPath)
			if err != nil {
				return fmt.Errorf("unable to create : %s", err)
			}
		}

		isFirstSection := false
		allRegistries, err := GetProfileRegistries()
		if err != nil {
			return err
		}

		if len(allRegistries) == 0 {
			isFirstSection = true
		}

		awsConfigFile, filepath, err := loadAWSConfigFile()
		if err != nil {
			return err
		}

		if err := Sync(&registry, awsConfigFile, syncOpts{
			isFirstSection:                 isFirstSection,
			promptUserIfProfileDuplication: true,
			shouldSilentLog:                false,
			shouldFailForRequiredKeys:      false,
		}); err != nil {
			return err
		}

		// reload the config.
		gConf, err = grantedConfig.Load()
		if err != nil {
			return err
		}

		// we have verified that this registry is a valid one and sync is completed.
		// so save the new registry to config file.
		gConf.ProfileRegistry.Registries = append(gConf.ProfileRegistry.Registries, registry.Config)
		err = gConf.Save()
		if err != nil {
			return err
		}

		// if priority flag is provided
		if priority > 0 {
			allRegistries, err = GetProfileRegistries()
			if err != nil {
				return err
			}

			// since the list is stored in sorted order. Only checking the second item suffice.
			// Error here try to fix it.
			if len(allRegistries) > 1 {
				var currentHighest int = 0
				if allRegistries[1].Config.Priority != nil {
					currentHighest = *allRegistries[1].Config.Priority
				}

				// this means that the new registry has the highest prority i.e
				// these profiles shouldn't have registry name as duplication.
				if currentHighest < priority {
					err = removeAutogeneratedProfiles(awsConfigFile, filepath)
					if err != nil {
						return err
					}

					clio.Debugf("New Registry has higher priority, resyncing all registries in new order...")
					err = SyncProfileRegistries(false, false, false)
					if err != nil {
						return err
					}

					return nil
				}
			}
		}

		err = awsConfigFile.SaveTo(filepath)
		if err != nil {
			return err
		}

		return nil
	},
}

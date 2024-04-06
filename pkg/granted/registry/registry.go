package registry

import (
	"context"
	"fmt"
	"sort"

	grantedConfig "github.com/common-fate/granted/pkg/config"
	"github.com/common-fate/granted/pkg/granted/registry/gitregistry"
	"github.com/common-fate/granted/pkg/granted/registry/rpcregistry"
	"gopkg.in/ini.v1"
)

type Registry interface {
	AWSProfiles(ctx context.Context) (*ini.File, error)
}

type loadedRegistry struct {
	Config   grantedConfig.Registry
	Registry Registry
}

func GetProfileRegistries(interactive bool) ([]loadedRegistry, error) {
	gConf, err := grantedConfig.Load()
	if err != nil {
		return nil, err
	}

	if len(gConf.ProfileRegistry.Registries) == 0 {
		return []loadedRegistry{}, nil
	}

	var registries []loadedRegistry
	for _, r := range gConf.ProfileRegistry.Registries {
		var reg Registry

		switch r.Type {
		case "git", "": // empty string here ensures backwards compatibility
			reg, err = gitregistry.New(gitregistry.Opts{
				Name:        r.Name,
				URL:         r.URL,
				Path:        r.Path,
				Filename:    r.Filename,
				Interactive: interactive,
			})
		case "commonfate.access.v1alpha1":
			reg = rpcregistry.Registry{}

		default:
			filepath, loadErr := grantedConfig.GrantedConfigFilePath()
			if loadErr != nil {
				filepath = "<error loading filepath>"
			}

			err = fmt.Errorf("invalid registry type: %s. registry type must be one of: ['git', 'commonfate.access.v1alpha1']. you can edit the registry config in your Granted config file (%s) to fix this", r.Type, filepath)
		}
		if err != nil {
			return nil, err
		}

		registries = append(registries, loadedRegistry{
			Config:   r,
			Registry: reg,
		})
	}

	// this will sort the registry based on priority.
	sort.Slice(registries, func(i, j int) bool {
		a := registries[i].Config.Priority
		b := registries[j].Config.Priority

		return a > b
	})

	return registries, nil
}

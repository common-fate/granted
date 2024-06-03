package registry

import (
	"context"
	"sort"

	grantedConfig "github.com/common-fate/granted/pkg/config"
	"github.com/common-fate/granted/pkg/granted/registry/cfregistry"
	"github.com/common-fate/granted/pkg/granted/registry/gitregistry"
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

		if r.Type == "git" || r.Type == "" {
			reg, err := gitregistry.New(gitregistry.Opts{
				Name:        r.Name,
				URL:         r.URL,
				Path:        r.Path,
				Filename:    r.Filename,
				Interactive: interactive,
			})

			if err != nil {
				return nil, err
			}
			registries = append(registries, loadedRegistry{
				Config:   r,
				Registry: reg,
			})
		} else {
			//set up a common fate registry
			reg := cfregistry.New(cfregistry.Opts{
				Name: r.Name,
				URL:  r.URL,
			})
			registries = append(registries, loadedRegistry{
				Config:   r,
				Registry: reg,
			})
		}

	}

	// this will sort the registry based on priority.
	sort.Slice(registries, func(i, j int) bool {
		a := registries[i].Config.Priority
		b := registries[j].Config.Priority

		return a > b
	})

	return registries, nil
}

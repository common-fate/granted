package registry

import (
	"context"
	"regexp"
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

func IsGitRepository(url string) bool {
	regex := regexp.MustCompile(`(?:https:\/\/github\.com\/|git@github\.com:)[a-zA-Z0-9_-]+\/[a-zA-Z0-9_-]+(?:\.git)?`)

	return regex.MatchString(url)
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

		if IsGitRepository(r.URL) {
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
			reg, err := cfregistry.New(cfregistry.Opts{
				Name: r.Name,
				URL:  r.URL,
			})

			if err != nil {
				return nil, err
			}
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

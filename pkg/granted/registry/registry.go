package registry

import (
	"os"
	"path"

	"github.com/common-fate/clio"
	grantedConfig "github.com/common-fate/granted/pkg/config"
	"gopkg.in/yaml.v3"
)

type Registry struct {
	AwsConfigPaths []string `yaml:"awsConfig"`
	Variables      []string `yaml:"variables"`
	Config         grantedConfig.Registry
}

// GetRegistryLocation returns the directory path where cloned repo is located.
func (r *Registry) getRegistryLocation() (string, error) {
	gConfigPath, err := grantedConfig.GrantedConfigFolder()
	if err != nil {
		return "", err
	}

	return path.Join(gConfigPath, "registries", r.Config.Name), nil
}

func (r *Registry) Parse() error {
	var configFilePath string = "granted.yml"
	if r.Config.Path != nil {
		configFilePath = *r.Config.Path
	}

	filepath, err := r.getRegistryLocation()
	if err != nil {
		return err
	}

	grantedFilePath := path.Join(filepath, configFilePath)
	clio.Debugf("verifying if valid config exists in %s", grantedFilePath)
	file, err := os.ReadFile(grantedFilePath)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(file, r)
	if err != nil {
		return err
	}

	return nil
}

func NewProfileRegistry(name string, url string) Registry {
	return Registry{
		Config: grantedConfig.Registry{
			Name: name,
			URL:  url,
		},
	}
}

func GetProfileRegistries() ([]Registry, error) {
	gConf, err := grantedConfig.Load()
	if err != nil {
		return nil, err
	}

	if len(gConf.ProfileRegistry.Registries) == 0 {
		return []Registry{}, nil
	}

	registries := []Registry{}
	for _, item := range gConf.ProfileRegistry.Registries {
		registries = append(registries, Registry{
			Config: grantedConfig.Registry{
				Name: item.Name,
				URL:  item.URL,
			},
		})
	}

	return registries, nil
}

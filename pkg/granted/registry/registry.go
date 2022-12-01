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
	const defaultConfigFilename string = "granted.yml"

	filepath, err := r.getRegistryLocation()
	if err != nil {
		return err
	}

	if r.Config.Path != nil {
		filepath = path.Join(filepath, *r.Config.Path)
	}

	fileInfo, err := os.Stat(filepath)
	if err != nil {
		return err
	}

	// if the provided path is a directory then
	// we will assume that it has default config file i.e `granted.yml` file inside the given directory.
	if fileInfo.IsDir() {
		filepath = path.Join(filepath, defaultConfigFilename)
	}

	clio.Debugf("verifying if valid config exists in %s", filepath)
	file, err := os.ReadFile(filepath)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(file, r)
	if err != nil {
		return err
	}

	return nil
}

type registryOptions struct {
	name     string
	path     string
	ref      string
	url      string
	priority int
}

func NewProfileRegistry(rOpts registryOptions) Registry {
	newRegistry := Registry{
		Config: grantedConfig.Registry{
			Name: rOpts.name,
			URL:  rOpts.url,
		},
	}

	if rOpts.path != "" {
		newRegistry.Config.Path = &rOpts.path
	}

	if rOpts.ref != "" {
		newRegistry.Config.Ref = &rOpts.ref
	}

	return newRegistry
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

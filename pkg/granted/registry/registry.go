package registry

import (
	"os"
	"path"

	"github.com/common-fate/granted/pkg/config"
	"gopkg.in/yaml.v3"
)

type Registry struct {
	AwsConfigPaths []string `yaml:"awsConfig"`
}

// Parse the `granted.yml` file.
func (c *Registry) Parse(folderpath string) (*Registry, error) {
	file, err := os.ReadFile(path.Join(folderpath, "granted.yml"))
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(file, c)
	if err != nil {
		return nil, err
	}

	return c, nil
}

// GetRegistryLocation returns the directory path where cloned repo is located.
func getRegistryLocation(u GitURL) (string, error) {
	gConfigPath, err := config.GrantedConfigFolder()
	if err != nil {
		return "", err
	}

	return path.Join(gConfigPath, "registries", u.Host, (u.Org + "/" + u.Repo)), nil
}

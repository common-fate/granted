package registry

import (
	"os"
	"path"

	"github.com/common-fate/clio"
	"github.com/common-fate/granted/pkg/config"
	"gopkg.in/yaml.v3"
)

type Registry struct {
	AwsConfigPaths []string `yaml:"awsConfig"`
	Url            GitURL
}

// Parse the config file if provided or 'granted.yml' file by default.
func (c *Registry) Parse(folderpath string, url GitURL) (*Registry, error) {
	var configFileName string = "granted.yml"
	if url.Filename != "" {
		configFileName = url.Filename
	}

	grantedFilePath := path.Join(folderpath, url.Subpath, configFileName)
	clio.Debugf("reading awsConfigs listed in %s", grantedFilePath)
	file, err := os.ReadFile(grantedFilePath)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(file, c)
	if err != nil {
		return nil, err
	}

	c.Url = url

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

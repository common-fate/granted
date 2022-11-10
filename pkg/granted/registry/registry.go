package registry

import (
	"os"
	"path"
	"strings"

	"github.com/common-fate/granted/pkg/config"
	"gopkg.in/yaml.v3"
)

type Registry struct {
	AwsConfigPaths []string `yaml:"awsConfig"`
	Subpath        string
}

// Parse the `granted.yml` file.
func (c *Registry) Parse(folderpath string, url GitURL) (*Registry, error) {
	var grantedFilePath string
	if url.Subpath != "" {
		c.Subpath = url.Subpath

		// the subpath specifies granted.yml
		if strings.Contains(url.Subpath, "granted.yml") || strings.Contains(url.Subpath, "granted.yaml") {
			grantedFilePath = path.Join(folderpath, url.Subpath)
		} else {
			grantedFilePath = path.Join(folderpath, url.Subpath, "granted.yml")
		}

	} else {
		grantedFilePath = path.Join(folderpath, "granted.yml")
	}

	file, err := os.ReadFile(grantedFilePath)
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

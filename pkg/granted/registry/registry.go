package registry

import (
	"net/url"
	"os"
	"path"

	"github.com/common-fate/granted/pkg/config"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

type Registry struct {
	AwsConfigPaths []string `yaml:"awsConfig"`
}

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

func GetRegistryLocation(u *url.URL) (string, error) {
	gConfigPath, err := config.GrantedConfigFolder()
	if err != nil {
		return "", err
	}

	return path.Join(gConfigPath, "registries", u.Host, formatFolderPath(u.Path)), nil
}

var ProfileRegistry = cli.Command{
	Name:        "registry",
	Subcommands: []*cli.Command{&AddCommand, &SyncCommand},
}

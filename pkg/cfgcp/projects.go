package cfgcp

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	resourcemanager "cloud.google.com/go/resourcemanager/apiv3"
	"google.golang.org/api/iterator"
	"gopkg.in/ini.v1"

	resourcemanagerpb "cloud.google.com/go/resourcemanager/apiv3/resourcemanagerpb"
)

type GCPConfig struct {
	Name     string
	IsActive bool
	Account  string `ini:"account"`
	Project  string `ini:"project"`
	Zone     string `ini:"zone"`   //todo type this
	Region   string `ini:"region"` //todo type this
}

type GCPLoader struct {
}

const (
	OSX_PATH     = "/.config/gcloud"
	WINDOWS_PATH = `%APPDATA%\gcloud`
	LINUX_PATH   = "/.config/gcloud"
)

func (g *GCPLoader) GetOSSpecifcConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	switch runtime.GOOS {
	case "windows":
		return home + WINDOWS_PATH, nil
	case "darwin":
		return home + OSX_PATH, nil
	case "linux":
		return home + LINUX_PATH, nil
	default:
		return "", errors.New("os not supported")
	}

}

// reads all config files for their names in ~/.config/gcloud
func (g *GCPLoader) Load() ([]string, error) {
	configs := []string{}
	configLocation, err := g.GetOSSpecifcConfigPath()
	if err != nil {
		return nil, err
	}

	err = filepath.WalkDir(path.Join(configLocation, "configurations"), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasPrefix(d.Name(), "config_") {
			configs = append(configs, d.Name()[7:])
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return configs, nil
}

// reads all config files for their names in ~/.config/gcloud
func (g *GCPLoader) Get(configId string) (GCPConfig, error) {
	config := GCPConfig{}

	configLocation, err := g.GetOSSpecifcConfigPath()
	if err != nil {
		return config, err
	}

	selectedConfigFilePath := path.Join(configLocation, "configurations", fmt.Sprintf("/config_%s", configId))
	coreConfig, err := ini.LoadSources(ini.LoadOptions{}, selectedConfigFilePath)
	if err != nil {
		return config, err
	}
	core, err := coreConfig.GetSection("core")
	if err != nil {
		return config, err
	}
	err = core.MapTo(&config)
	if err != nil {
		return config, err
	}
	return config, nil
}

func ListProjects(ctx context.Context) ([]resourcemanagerpb.Project, error) {
	// This snippet has been automatically generated and should be regarded as a code template only.
	// It will require modifications to work:
	// - It may require correct/in-range values for request initialization.
	// - It may require specifying regional endpoints when creating the service client as shown in:
	//   https://pkg.go.dev/cloud.google.com/go#hdr-Client_Options
	c, err := resourcemanager.NewProjectsClient(ctx)
	if err != nil {
		return nil, err
	}
	defer c.Close()

	req := &resourcemanagerpb.ListProjectsRequest{
		// TODO: Fill request struct fields.
		// See https://pkg.go.dev/cloud.google.com/go/resourcemanager/apiv3/resourcemanagerpb#ListProjectsRequest.
		Parent: "organizations/892941281001",
	}

	it := c.ListProjects(ctx, req)
	for {
		resp, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		// TODO: Use resp.
		fmt.Printf("%s", resp.ProjectId)
	}
	return nil, nil
}

package cfgcp

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	resourcemanager "cloud.google.com/go/resourcemanager/apiv3"
	"google.golang.org/api/iterator"

	resourcemanagerpb "cloud.google.com/go/resourcemanager/apiv3/resourcemanagerpb"
)

type GCPConfig struct {
	Name          string
	isActive      bool
	Account       string
	Project       string
	DefaultZone   string //todo type this
	DefaultRegion string //todo type this
}

type GCPLoader struct {
}

const OSX_PATH = "/.config/gcloud/configurations"
const WINDOWS_PATH = `%APPDATA%\gcloud\configurations`
const LINUX_PATH = "/.config/gcloud/configurations"

func (g *GCPLoader) getOSConfigLocation() (string, error) {
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
	configLocation, err := g.getOSConfigLocation()
	if err != nil {
		return nil, err
	}

	err = filepath.WalkDir(configLocation, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && d.Name() != "" {
			configs = append(configs, d.Name()[7:])
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return configs, nil
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

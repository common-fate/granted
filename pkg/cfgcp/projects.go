package cfgcp

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/ini.v1"
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

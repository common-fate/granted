package registry

import (
	"os"
	"path/filepath"

	"gopkg.in/ini.v1"
)

// Find the ~/.aws/config absolute path based on OS.
func getDefaultAWSConfigLocation() (string, error) {
	h, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	configPath := filepath.Join(h, ".aws", "config")
	return configPath, nil
}

func loadAWSConfigFile() (*ini.File, string, error) {
	filepath, err := getDefaultAWSConfigLocation()
	if err != nil {
		return nil, "", err
	}

	awsConfig, err := loadAWSConfigFileFromPath(filepath)
	if err != nil {
		return nil, "", err
	}
	return awsConfig, filepath, nil
}

func loadAWSConfigFileFromPath(filepath string) (*ini.File, error) {
	awsConfig, err := ini.LoadSources(ini.LoadOptions{
		SkipUnrecognizableLines: true,
		AllowNonUniqueSections:  true,
		AllowNestedValues:       true,
	}, filepath)
	if err != nil {
		return nil, err
	}

	return awsConfig, nil
}

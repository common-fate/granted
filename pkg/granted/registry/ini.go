package registry

import (
	"os"
	"path"
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

func loadAWSConfigFile() (*ini.File, error) {
	p, err := getDefaultAWSConfigLocation()
	if err != nil {
		return nil, err
	}

	awsConfig, err := ini.LoadSources(ini.LoadOptions{
		SkipUnrecognizableLines: true,
		AllowNonUniqueSections:  true,
	}, p)
	if err != nil {
		return nil, err
	}

	return awsConfig, nil
}

// load all cloned configs of a single repo into one ini object.
// this will overwrite if there are duplicate profiles with same name.
func loadClonedConfigs(r Registry, repoDirPath string) (*ini.File, error) {
	clonedFile := ini.Empty()
	for _, cfile := range r.AwsConfigPaths {
		filepath := path.Join(repoDirPath, cfile)

		err := clonedFile.Append(filepath)
		if err != nil {
			return nil, err
		}
	}

	return clonedFile, nil
}

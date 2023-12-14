package registry

import (
	"os"
	"path"
	"path/filepath"

	"github.com/common-fate/clio"
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

// load all cloned configs of a single repo into one ini object.
// this will overwrite if there are duplicate profiles with same name.
func loadClonedConfigs(r Registry) (*ini.File, error) {
	clonedFile := ini.Empty()

	repoDirPath, err := getRegistryLocation(r.Config)
	if err != nil {
		return nil, err
	}

	for _, cfile := range r.AwsConfigPaths {
		var filepath string
		if r.Config.Path != nil {
			filepath = path.Join(repoDirPath, *r.Config.Path, cfile)
		} else {
			filepath = path.Join(repoDirPath, cfile)
		}

		clio.Debugf("loading aws config file from %s", filepath)
		err := clonedFile.Append(filepath)
		if err != nil {
			return nil, err
		}
	}

	return clonedFile, nil
}

package cfaws

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/bigkevmcd/go-configparser"
)

func GetProfilesFromDefaultSharedConfig() ([]string, error) {

	// fetch the parsed config file
	configPath := config.DefaultSharedConfigFilename()
	config, err := configparser.NewConfigParserFromFile(configPath)
	if err != nil {
		return nil, err
	}

	var profileMap = make(map[string]string)
	allProfileOptions := []string{}

	// .aws/config files are structured as follows,
	// We want to strip the profile_name i.e. [profile <profile_name>],
	//
	// [profile cf-dev]
	// sso_region=ap-southeast-2
	// ...
	// [profile cf-prod]
	// sso_region=ap-southeast-2
	// ...

	// Itterate through the config sections
	for _, section := range config.Sections() {

		// Check if the section is prefixed with 'profile '
		if section[0:7] == "profile" {

			// Strip 'profile ' from the section name
			awsProfile := section[8:]
			ssoID, err := config.Get(section, "sso_account_id")
			if err != nil {
				return nil, err
			}

			value := fmt.Sprintf("%-16s%s:%s", awsProfile, "aws", ssoID)

			profileMap[awsProfile] = value
			allProfileOptions = append(allProfileOptions, value)
		}
	}
	return allProfileOptions, nil
}

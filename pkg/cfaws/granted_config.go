package cfaws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/bigkevmcd/go-configparser"
)

type grantedSSOConfig struct {
	granted_sso_start_url    string
	granted_sso_region       string
	granted_sso_account_name string
	granted_sso_account_id   string
	granted_sso_role_name    string
	region                   string
}

func NewGrantedConfig(rawConfig configparser.Dict) *grantedSSOConfig {
	return &grantedSSOConfig{
		granted_sso_start_url:    rawConfig["granted_sso_start_url"],
		granted_sso_region:       rawConfig["granted_sso_region"],
		granted_sso_account_name: rawConfig["granted_sso_account_name"],
		granted_sso_account_id:   rawConfig["granted_sso_account_id"],
		granted_sso_role_name:    rawConfig["granted_sso_role_name"],
		region:                   rawConfig["region"],
	}
}

func (gConfig *grantedSSOConfig) ConvertToAWSConfig(ctx context.Context, p *Profile) (*config.SharedConfig, error) {
	cfg, err := config.LoadSharedConfigProfile(ctx, p.Name, func(lsco *config.LoadSharedConfigOptions) { lsco.ConfigFiles = []string{p.File} })
	// if required profile doesn't exist then
	// return empty config instead of error.
	if err != nil {
		if _, ok := err.(config.SharedConfigProfileNotExistError); ok {
			return &config.SharedConfig{}, nil
		}
		return nil, err
	}

	cfg.SSOAccountID = gConfig.granted_sso_account_id
	cfg.SSORegion = gConfig.granted_sso_region
	cfg.SSORoleName = gConfig.granted_sso_role_name
	cfg.SSOStartURL = gConfig.granted_sso_start_url
	cfg.Region = gConfig.region

	return &cfg, err
}

// For `granted login` cmd, we have to make sure 'granted' prefix
// is added to the aws config file.
func IsValidGrantedProfile(rawConfig configparser.Dict) error {
	requiredGrantedCredentials := []string{"granted_sso_start_url", "granted_sso_region", "granted_sso_account_id", "granted_sso_role_name", "region"}

	for _, value := range requiredGrantedCredentials {
		if _, ok := rawConfig[value]; !ok {
			return fmt.Errorf("invalid aws config for granted login. %s is undefined but necessary", value)
		}
	}

	return nil
}

// check if the passed aws config consist of "granted-sso-start-url" key.
func hasGrantedPrefix(rawConfig configparser.Dict) bool {
	if _, ok := rawConfig["granted_sso_start_url"]; ok {
		return true
	}

	return false
}

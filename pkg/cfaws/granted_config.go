package cfaws

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/common-fate/clio"
	"gopkg.in/ini.v1"
)

func ParseGrantedSSOProfile(ctx context.Context, profile *Profile) (*config.SharedConfig, error) {
	err := IsValidGrantedProfile(profile.RawConfig)
	if err != nil {
		return nil, err
	}
	cfg, err := config.LoadSharedConfigProfile(ctx, profile.Name, func(lsco *config.LoadSharedConfigOptions) { lsco.ConfigFiles = []string{profile.File} })
	if err != nil {
		return nil, err
	}
	item, err := profile.RawConfig.GetKey("granted_sso_account_id")
	if err != nil {
		return nil, err
	}
	cfg.SSOAccountID = item.Value()
	item, err = profile.RawConfig.GetKey("granted_sso_region")
	if err != nil {
		return nil, err
	}
	cfg.SSORegion = item.Value()
	item, err = profile.RawConfig.GetKey("granted_sso_role_name")
	if err != nil {
		return nil, err
	}
	cfg.SSORoleName = item.Value()
	item, err = profile.RawConfig.GetKey("granted_sso_start_url")
	if err != nil {
		return nil, err
	}

	// sanity check to verify if the provided value is a valid url
	_, err = url.ParseRequestURI(item.Value())
	if err != nil {
		clio.Debug(err)
		return nil, fmt.Errorf("invalid value '%s' provided for 'granted_sso_start_url'", item.Value())
	}

	cfg.SSOStartURL = item.Value()
	return &cfg, err
}

// For `granted login` cmd, we have to make sure 'granted' prefix
// is added to the aws config file.
func IsValidGrantedProfile(rawConfig *ini.Section) error {
	requiredGrantedCredentials := []string{"granted_sso_start_url", "granted_sso_region", "granted_sso_account_id", "granted_sso_role_name"}
	for _, value := range requiredGrantedCredentials {
		if !rawConfig.HasKey(value) {
			return fmt.Errorf("invalid aws config for granted login. '%s' field must be provided", value)
		}
	}
	return nil
}

// check if the config section shas any keys prefixed with "granted_sso_"
func hasGrantedSSOPrefix(rawConfig *ini.Section) bool {
	for _, v := range rawConfig.KeyStrings() {
		if strings.HasPrefix(v, "granted_sso_") {
			return true
		}
	}
	return false
}

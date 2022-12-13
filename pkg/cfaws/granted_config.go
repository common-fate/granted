package cfaws

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/common-fate/clio"
	"gopkg.in/ini.v1"
)

func ParseGrantedSSOProfile(ctx context.Context, profile *Profile) (*config.SharedConfig, error) {
	err := IsValidGrantedProfile(profile)
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
		if profile.SSOSession != nil && profile.SSOSession.SSORegion != "" {
			cfg.SSORegion = profile.SSOSession.SSORegion
		} else {
			return nil, err
		}
	} else {
		cfg.SSORegion = item.Value()
	}

	item, err = profile.RawConfig.GetKey("granted_sso_role_name")
	if err != nil {
		return nil, err
	}
	cfg.SSORoleName = item.Value()

	item, err = profile.RawConfig.GetKey("granted_sso_start_url")
	if err != nil {
		if profile.SSOSession != nil && profile.SSOSession.SSORegion != "" {
			cfg.SSOStartURL = profile.SSOSession.SSOStartURL
		} else {
			return nil, err
		}
	} else {
		cfg.SSOStartURL = item.Value()
	}

	// sanity check to verify if the provided value is a valid url
	_, err = url.ParseRequestURI(cfg.SSOStartURL)
	if err != nil {
		clio.Debug(err)
		return nil, fmt.Errorf("invalid value '%s' provided for 'granted_sso_start_url'", cfg.SSOStartURL)
	}

	item, err = profile.RawConfig.GetKey("credential_process")
	if err != nil {
		return nil, err
	}

	err = validateCredentialProcess(item.Value(), profile.Name)
	if err != nil {
		return nil, err
	}

	return &cfg, err
}

// For `granted login` cmd, we have to make sure 'granted' prefix
// is added to the aws config file.
func IsValidGrantedProfile(profile *Profile) error {
	requiredGrantedCredentials := []string{"granted_sso_account_id", "granted_sso_role_name"} //"granted_sso_start_url", "granted_sso_region",
	for _, value := range requiredGrantedCredentials {
		if !profile.RawConfig.HasKey(value) {
			return fmt.Errorf("invalid aws config for granted login. '%s' field must be provided", value)
		}
	}
	if profile.SSOSession != nil {
		if profile.SSOSession.SSORegion == "" && !profile.RawConfig.HasKey("granted_sso_region") {
			return fmt.Errorf("invalid aws config for granted login. '%s' field must be provided", "granted_sso_region")
		}
		if profile.SSOSession.SSOStartURL == "" && !profile.RawConfig.HasKey("granted_sso_start_url") {
			return fmt.Errorf("invalid aws config for granted login. '%s' field must be provided", "granted_sso_start_url")
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

// validateCredentialProcess checks whether the granted_ prefixed AWS profiles
// are correctly using the granted credential-process override or not.
// also check whether the provided flag to 'granted credential-process --profile pname'
// matches the AWS config profile name. If it doesn't then return an err
// as the user will certainly run into unexpected behaviour.
func validateCredentialProcess(arg string, awsProfileName string) error {
	regex := regexp.MustCompile(`^(\s+)?(dgranted|granted)\s+credential-process.*--profile\s+(?P<PName>([^\s]+))`)

	if regex.MatchString(arg) {
		matches := regex.FindStringSubmatch(arg)
		pNameIndex := regex.SubexpIndex("PName")

		profileName := matches[pNameIndex]

		if profileName == "" {
			return fmt.Errorf("profile name not provided. Try adding profile name like 'granted credential-process --profile <profile-name>'")
		}

		// if matches then do nth.
		if profileName == awsProfileName {
			return nil
		}

		return fmt.Errorf("unmatched profile names. The profile name '%s' provided to 'granted credential-process' doesnot match AWS profile name '%s'", profileName, awsProfileName)
	}

	return fmt.Errorf("unable to parse 'credential_process'. Looks like your credential_process isn't configured correctly. \n You need to add 'granted credential-process --profile <profile-name>'")
}

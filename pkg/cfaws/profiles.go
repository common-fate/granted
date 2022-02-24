package cfaws

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/bigkevmcd/go-configparser"
	"github.com/common-fate/granted/pkg/debug"
	"github.com/pkg/errors"
)

type ProfileType int

const (
	ProfileTypeSSO ProfileType = iota
	ProfileTypeIAM
)

type CFSharedConfig struct {
	// the original config, some values may be empty strings depending on the type or profile
	RawConfig   config.SharedConfig
	Name        string
	ProfileType ProfileType
	// ordered from root to direct parent profile
	Parents []*CFSharedConfig
}
type CFSharedConfigs map[string]*CFSharedConfig

// GetProfilesFromDefaultSharedConfig returns initialised profiles
// these profiles state their type and parents
func GetProfilesFromDefaultSharedConfig(ctx context.Context) (CFSharedConfigs, error) {
	// fetch the parsed config file
	configPath := config.DefaultSharedConfigFilename()
	configFile, err := configparser.NewConfigParserFromFile(configPath)
	if err != nil {
		return nil, err
	}

	// .aws/config files are structured as follows,
	//
	// [profile cf-dev]
	// sso_region=ap-southeast-2
	// ...
	// [profile cf-prod]
	// sso_region=ap-southeast-2
	// ...
	profiles := make(map[string]*uninitCFSharedConfig)

	// Itterate through the config sections
	for _, section := range configFile.Sections() {
		// Check if the section is prefixed with 'profile ' and that the profile has a name
		if strings.HasPrefix(section, "profile ") && len(section) > 8 {
			name := strings.TrimPrefix(section, "profile ")
			cf, err := config.LoadSharedConfigProfile(ctx, name)
			if err != nil {
				debug.Fprintf(debug.VerbosityDebug, os.Stderr, "%s\n", errors.Wrap(err, "loading profiles from config").Error())
				continue
			} else {
				profiles[name] = &uninitCFSharedConfig{initialised: false, CFSharedConfig: &CFSharedConfig{RawConfig: cf, Name: name}}
			}
		}
	}

	// build parent trees and assert types
	for _, profile := range profiles {
		profile.init(profiles)
	}

	initialisedProfiles := make(map[string]*CFSharedConfig)
	for k, profile := range profiles {
		initialisedProfiles[k] = profile.CFSharedConfig
	}
	return initialisedProfiles, nil
}

// an interim type while the profiles are being initialised
type uninitCFSharedConfig struct {
	*CFSharedConfig
	initialised bool
}

func (c *uninitCFSharedConfig) init(profiles map[string]*uninitCFSharedConfig) {
	if !c.initialised {
		if c.RawConfig.SourceProfileName == "" {
			// this profile is a source for credentials
			if c.RawConfig.SSOAccountID != "" {
				c.ProfileType = ProfileTypeSSO
			} else {
				c.ProfileType = ProfileTypeIAM
			}
		} else {
			sourceProfile := profiles[c.RawConfig.SourceProfileName]
			sourceProfile.init(profiles)
			c.ProfileType = sourceProfile.ProfileType
			c.Parents = append(sourceProfile.Parents, sourceProfile.CFSharedConfig)
		}
		c.initialised = true
	}
}

// Region will attempt to load the reason on this profile, if it is not set, attempts to load the default config
// returns a region, and bool = true if the default region was used
func (c CFSharedConfig) Region(ctx context.Context) (string, bool, error) {
	defaultCfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return "", false, err
	}
	region := defaultCfg.Region
	if c.RawConfig.Region != "" {
		return c.RawConfig.Region, false, nil
	}
	if region == "" {
		return "", false, fmt.Errorf("region not set on profile %s, could not load a default AWS_REGION. Either set a default region `aws configure set default.region us-west-2` or add a region to your profile", c.Name)
	}
	return region, true, nil
}

func (c CFSharedConfigs) SSOProfileNames() []string {
	names := []string{}
	for k, v := range c {
		if v.ProfileType == ProfileTypeSSO {
			names = append(names, k)
		}
	}
	return names
}

func (c CFSharedConfigs) IAMProfileNames() []string {
	names := []string{}
	for k, v := range c {
		if v.ProfileType == ProfileTypeIAM {
			names = append(names, k)
		}
	}
	return names
}

func (c CFSharedConfigs) ProfileNames() []string {
	names := []string{}
	for k := range c {
		names = append(names, k)
	}
	return names
}

func (c *CFSharedConfig) AwsConfig(ctx context.Context, useSSORegion bool) (aws.Config, error) {

	opts := []func(*config.LoadOptions) error{
		// load the config profile
		config.WithSharedConfigProfile(c.Name),
	}

	if useSSORegion {
		// With region forces this config to use the profile region, ignoring region configured with environment variables
		opts = append(opts, config.WithRegion(c.RawConfig.SSORegion))
	} else if c.RawConfig.Region != "" {
		// With region forces this config to use the profile region, ignoring region configured with environment variables
		// if region is not configured for this profile, use the aws_default_region
		opts = append(opts, config.WithRegion(c.RawConfig.Region))
	}

	return config.LoadDefaultConfig(ctx,
		opts...,
	)
}

func (c *CFSharedConfig) CallerIdentity(ctx context.Context) (*sts.GetCallerIdentityOutput, error) {
	cfg, err := c.AwsConfig(ctx, false)
	if err != nil {
		return nil, err
	}
	client := sts.NewFromConfig(cfg)
	return client.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
}

func (c *CFSharedConfig) Assume(ctx context.Context) (aws.Credentials, error) {
	if c.ProfileType == ProfileTypeIAM {
		cfg, err := c.AwsConfig(ctx, false)
		if err != nil {
			return aws.Credentials{}, err
		}
		appCreds := aws.NewCredentialsCache(cfg.Credentials)
		return appCreds.Retrieve(ctx)
	} else {
		return c.SSOLogin(ctx)
	}
}

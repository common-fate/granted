package cfaws

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/bigkevmcd/go-configparser"
	"github.com/common-fate/granted/pkg/debug"
	"github.com/fatih/color"
	"github.com/pkg/errors"
)

type CFSharedConfig struct {
	// the original config, some values may be empty strings depending on the type or profile
	AWSConfig config.SharedConfig
	// allows access to the raw values from the file
	RawConfig   configparser.Dict
	Name        string
	ProfileType string
	// ordered from root to direct parent profile
	Parents []*CFSharedConfig
}
type CFSharedConfigs map[string]*CFSharedConfig

// GetProfilesFromDefaultSharedConfig returns initialised profiles
// these profiles state their type and parents
// The main reason we need to use a config parsing library here is to list the names of all the profiles.
// The aws SDK does not provide a method to list all profiles
//
// Secondary requirement is to identify profiles which use a specific credential process like saml2aws
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
		rawConfig, err := configFile.Items(section)
		if err != nil {
			debug.Fprintf(debug.VerbosityDebug, color.Error, "%s\n", errors.Wrap(err, "loading profiles from config").Error())
			continue
		}
		// Check if the section is prefixed with 'profile ' and that the profile has a name
		if strings.HasPrefix(section, "profile ") && len(section) > 8 {
			name := strings.TrimPrefix(section, "profile ")
			cf, err := config.LoadSharedConfigProfile(ctx, name)
			if err != nil {
				debug.Fprintf(debug.VerbosityDebug, color.Error, "%s\n", errors.Wrap(err, "loading profiles from config").Error())
				continue
			} else {
				profiles[name] = &uninitCFSharedConfig{initialised: false, CFSharedConfig: &CFSharedConfig{AWSConfig: cf, Name: name, RawConfig: rawConfig}}
			}
		}
	}

	// build parent trees and assert types
	for _, profile := range profiles {
		profile.init(profiles, 0)
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

func (c *uninitCFSharedConfig) init(profiles map[string]*uninitCFSharedConfig, depth int) {
	if !c.initialised {
		// Ensures this recursive call does not exceed a maximum depth
		// potentially triggered by bad config files with cycles in source_profile
		// In simple cases this seems to be picked up by the AWS sdk before the profiles are initialised which would log a debug message instead
		if depth < 10 {
			if c.AWSConfig.SourceProfileName == "" {
				as := assumers
				for _, a := range as {
					if a.ProfileMatchesType(c.RawConfig, c.AWSConfig) {
						c.ProfileType = a.Type()
						break
					}
				}
			} else {
				sourceProfile := profiles[c.AWSConfig.SourceProfileName]
				sourceProfile.init(profiles, depth+1)
				c.ProfileType = sourceProfile.ProfileType
				c.Parents = append(sourceProfile.Parents, sourceProfile.CFSharedConfig)
			}
		} else {
			fmt.Fprintf(color.Error, "maximum source profile depth exceeded for profile %s\nthis indicates that you have a cyclic reference in your aws profiles.[profile dev]\nregion = ap-southeast-2\nsource_profile = prod\n\n[profile prod]\nregion = ap-southeast-2\nsource_profile = dev", c.Name)
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
	if c.AWSConfig.Region != "" {
		return c.AWSConfig.Region, false, nil
	}
	if region == "" {
		return "", false, fmt.Errorf("region not set on profile %s, could not load a default AWS_REGION. Either set a default region `aws configure set default.region us-west-2` or add a region to your profile", c.Name)
	}
	return region, true, nil
}

// func (c CFSharedConfigs) SSOProfileNames() []string {
// 	names := []string{}
// 	for k, v := range c {
// 		if v.ProfileType == ProfileTypeSSO {
// 			names = append(names, k)
// 		}
// 	}
// 	return names
// }

// func (c CFSharedConfigs) IAMProfileNames() []string {
// 	names := []string{}
// 	for k, v := range c {
// 		if v.ProfileType == ProfileTypeIAM {
// 			names = append(names, k)
// 		}
// 	}
// 	return names
// }

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
		opts = append(opts, config.WithRegion(c.AWSConfig.SSORegion))
	} else if c.AWSConfig.Region != "" {
		// With region forces this config to use the profile region, ignoring region configured with environment variables
		// if region is not configured for this profile, use the aws_default_region
		opts = append(opts, config.WithRegion(c.AWSConfig.Region))
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

func (c *CFSharedConfig) AssumeConsole(ctx context.Context, args []string) (aws.Credentials, error) {
	return AssumerFromType(c.ProfileType).AssumeConsole(ctx, c, args)
}

func (c *CFSharedConfig) AssumeTerminal(ctx context.Context, args []string) (aws.Credentials, error) {
	return AssumerFromType(c.ProfileType).AssumeTerminal(ctx, c, args)
}

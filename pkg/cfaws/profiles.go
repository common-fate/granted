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
				if os.Getenv("DEBUG") == "true" {
					fmt.Fprintln(os.Stderr, err.Error())
				}

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

func (c *CFSharedConfig) AwsConfig(ctx context.Context) (aws.Config, error) {
	return config.LoadDefaultConfig(ctx,
		config.WithSharedConfigProfile(c.Name),
	)
}

func (c *CFSharedConfig) CallerIdentity(ctx context.Context) (*sts.GetCallerIdentityOutput, error) {
	cfg, err := c.AwsConfig(ctx)
	if err != nil {
		return nil, err
	}
	client := sts.NewFromConfig(cfg)
	return client.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
}

func (c *CFSharedConfig) Assume(ctx context.Context) (aws.Credentials, error) {
	if c.ProfileType == ProfileTypeIAM {
		cfg, err := c.AwsConfig(ctx)
		if err != nil {
			return aws.Credentials{}, err
		}
		appCreds := aws.NewCredentialsCache(cfg.Credentials)
		return appCreds.Retrieve(ctx)
	} else {
		return c.SSOLogin(ctx)
	}
}

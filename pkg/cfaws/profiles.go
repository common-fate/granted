package cfaws

import (
	"context"
	"os"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/bigkevmcd/go-configparser"
	"github.com/common-fate/granted/pkg/debug"
	"github.com/common-fate/granted/pkg/frecency"
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
		// load the config profile
		config.WithSharedConfigProfile(c.Name),
		// With region forces this config to use the profile region, ignoring region configured with environment variables
		config.WithRegion(c.RawConfig.Region),
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

var frecencyStoreKey = "aws_profiles_frecency"

type FrecentProfiles struct {
	store    *frecency.FrecencyStore
	toRemove []string
}

// should be called after selecting a profile to update frecency cache
// wrap this method in a go routine to avoid blocking the user
func (f *FrecentProfiles) Update(selectedProfile string) {
	s := make([]interface{}, len(f.toRemove))
	for i, v := range f.toRemove {
		s[i] = v
	}
	err := f.store.DeleteAll(s)
	if err != nil {
		debug.Fprintf(debug.VerbosityDebug, os.Stderr, errors.Wrap(err, "removing entries from frecency").Error())
	}
	err = f.store.Upsert(selectedProfile)
	if err != nil {
		debug.Fprintf(debug.VerbosityDebug, os.Stderr, errors.Wrap(err, "upserting entry to frecency").Error())
	}
}

// use this to update frecency cache when the profile is supplied by the commandline
func UpdateFrecencyCache(selectedProfile string) {
	fr, err := frecency.Load(frecencyStoreKey)
	if err != nil {
		debug.Fprintf(debug.VerbosityDebug, os.Stderr, errors.Wrap(err, "loading aws_profiles_frecency frecency store").Error())
	} else {
		err = fr.Upsert(selectedProfile)
		if err != nil {
			debug.Fprintf(debug.VerbosityDebug, os.Stderr, errors.Wrap(err, "upserting entry to frecency").Error())
		}
	}
}

// loads the frecency cache and generates a list of profiles with frecently used profiles first, followed by alphabetically sorted profiles that have not been used with assume
// this method returns a FrecentProfiles pointer which should be used after selecting a profile to update the cache, it will also remove any entries which no longer exist in the aws config
func (c CFSharedConfigs) GetFrecentProfiles() (*FrecentProfiles, []string) {
	names := []string{}
	namesMap := make(map[string]string)
	profileNames := c.ProfileNames()
	pnMap := make(map[string]string)
	for _, pn := range profileNames {
		pnMap[pn] = pn
	}
	fr, err := frecency.Load(frecencyStoreKey)
	if err != nil {
		debug.Fprintf(debug.VerbosityDebug, os.Stderr, errors.Wrap(err, "loading aws_profiles_frecency frecency store").Error())
	}
	namesToRemoveFromFrecency := []string{}

	// add all frecent profile names in order if they are still present in the profileNames slice
	for _, entry := range fr.Entries {
		e := entry.Entry.(string)
		if _, ok := pnMap[e]; ok {
			names = append(names, e)
			namesMap[e] = e
		} else {
			namesToRemoveFromFrecency = append(namesToRemoveFromFrecency, e)
		}
	}

	// add all other entries from profileNames, sort them alphabetically first
	namesToSort := []string{}
	for _, n := range profileNames {
		if _, ok := namesMap[n]; !ok {
			namesToSort = append(namesToSort, n)
		}
	}
	sort.Strings(namesToSort)
	names = append(names, namesToSort...)

	frPr := &FrecentProfiles{store: fr, toRemove: namesToRemoveFromFrecency}

	return frPr, names
}

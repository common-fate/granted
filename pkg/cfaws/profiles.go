package cfaws

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/ssocreds"
	"github.com/common-fate/clio"
	"github.com/common-fate/granted/internal/build"
	"gopkg.in/ini.v1"
)

type ConfigOpts struct {
	Duration time.Duration
	Args     []string
}

type SSOSession struct {
	SSORegion   string
	SSOStartURL string
}

type Profile struct {
	// allows access to the raw values from the file
	RawConfig *ini.Section
	Name      string
	// the file that this profile is from
	File        string
	ProfileType string

	// ordered from root to direct parent profile
	Parents []*Profile
	// the original config, some values may be empty strings depending on the type or profile
	AWSConfig                      config.SharedConfig
	Initialised                    bool
	LoadingError                   error
	HasSecureStorageIAMCredentials bool

	// AWS SDK doesn't support sso_session yet so we check for it manually
	SSOSession *SSOSession
}

var ErrProfileNotInitialised error = errors.New("profile not initialised")

var ErrProfileNotFound error = errors.New("profile not found")

type Profiles struct {
	// alphabetically sorted after first load
	ProfileNames []string
	profiles     map[string]*Profile
}

func LoadSSOSessions(configFile *ini.File) (map[string]SSOSession, error) {
	sessions := make(map[string]SSOSession)

	// Iterate through the config sections
	for _, section := range configFile.Sections() {
		// the ini package adds an extra section called DEFAULT, but this is different to the AWS standard of 'default' so we ignore it and only look at 'default'
		if strings.HasPrefix(section.Name(), "sso-session ") {
			session := SSOSession{}
			regionKey, err := section.GetKey("sso_region")
			if err == nil {
				session.SSORegion = regionKey.Value()
			}
			startURLKey, err := section.GetKey("sso_start_url")
			if err == nil {
				session.SSOStartURL = startURLKey.Value()
			}
			sessions[strings.TrimPrefix(section.Name(), "sso-session ")] = session
		}
	}
	return sessions, nil
}
func (p *Profiles) HasProfile(profile string) bool {
	_, ok := p.profiles[profile]
	return ok
}

// if the profile has a "granted_${name}" key, the value is returned. else an empty string
func (p *Profile) CustomGrantedProperty(name string) string {
	// rawConfig can be nil when all the required parameters are passed as arguments
	// like assume --sso --sso_start-url ...
	if p.RawConfig == nil {
		return ""
	}

	key, err := p.RawConfig.GetKey(fmt.Sprintf("granted_%s", name))
	if err != nil {
		return ""
	}
	return key.Value()
}
func (p *Profiles) Profile(profile string) (*Profile, error) {
	if c, ok := p.profiles[profile]; ok {
		return c, nil
	}
	return nil, ErrProfileNotFound
}

type ConfigFileLoader interface {
	Load() (*ini.File, error)
	Path() string
}

type FileLoader struct {
	FilePath string
}

func (f FileLoader) Path() string {
	return f.FilePath
}
func (f FileLoader) Load() (*ini.File, error) {
	configFile, err := ini.LoadSources(ini.LoadOptions{
		AllowNonUniqueSections:  false,
		SkipUnrecognizableLines: false,
	}, f.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return ini.Empty(), nil
		}
		return nil, err
	}
	return configFile, nil
}
func LoadProfilesFromDefaultFiles() (*Profiles, error) {
	return LoadProfiles(FileLoader{
		FilePath: config.DefaultSharedConfigFilename(),
	}, FileLoader{
		FilePath: config.DefaultSharedCredentialsFilename(),
	})
}
func LoadProfiles(configFileLoader, credentialsFileLoader ConfigFileLoader) (*Profiles, error) {

	p := Profiles{profiles: make(map[string]*Profile)}
	err := p.loadDefaultConfigFile(configFileLoader)
	if err != nil {
		return nil, err
	}
	err = p.loadDefaultCredentialsFile(credentialsFileLoader)
	if err != nil {
		return nil, err
	}
	sort.Strings(p.ProfileNames)
	return &p, nil
}

// .aws/config files are structured as follows,
//
// [profile cf-dev]
// sso_region=ap-southeast-2
// ...
// [profile cf-prod]
// sso_region=ap-southeast-2
// ...
func (p *Profiles) loadDefaultConfigFile(loader ConfigFileLoader) error {

	configFile, err := loader.Load()
	if err != nil {
		return err
	}
	ssoSessions, err := LoadSSOSessions(configFile)
	if err != nil {
		return err
	}
	// Iterate through the config sections
	for _, section := range configFile.Sections() {
		// the ini package adds an extra section called DEFAULT, but this is different to the AWS standard of 'default' so we ignore it an only look at 'default'
		if section.Name() != "DEFAULT" {
			// Check if the section is prefixed with 'profile ' and that the profile has a name
			if ((strings.HasPrefix(section.Name(), "profile ") && len(section.Name()) > 8) || section.Name() == "default") && IsLegalProfileName(strings.TrimPrefix(section.Name(), "profile ")) {
				name := strings.TrimPrefix(section.Name(), "profile ")
				sectionPtr := section
				profile := &Profile{RawConfig: sectionPtr, Name: name, File: loader.Path()}
				if section.HasKey("sso_session") {
					key, _ := section.GetKey("sso_session")
					ssoSession, ok := ssoSessions[key.Value()]
					if !ok {
						clio.Errorf("failed to load config profile %s because it has an 'sso_session = %s' section but the [sso-session %s] section was not found", name, key.Value(), key.Value())
						continue
					}
					profile.SSOSession = &ssoSession
				}
				p.ProfileNames = append(p.ProfileNames, name)
				p.profiles[name] = profile
			}
		}
	}
	return nil
}

// .aws/configuration files are structured as follows,
//
// [cf-dev]
// aws_access_key_id = xxxxxx
// aws_secret_access_key = xxxxxx
// ...
// [cf-prod]
// aws_access_key_id = xxxxxx
// aws_secret_access_key = xxxxxx
// ...
func (p *Profiles) loadDefaultCredentialsFile(loader ConfigFileLoader) error {
	// fetch parsed credentials file
	credentialsFile, err := loader.Load()
	if err != nil {
		return err
	}
	for _, section := range credentialsFile.Sections() {
		// the ini package adds an extra section called DEFAULT, but this is different to the AWS standard of 'default' so we ignore it and only look at 'default'
		if section.Name() != "DEFAULT" {
			// We only care about the non default sections for the credentials file (no profile prefix either)
			if section.Name() != "default" && IsLegalProfileName(section.Name()) {
				// check for a duplicate profile in the map and skip if present (config file should take precedence)
				_, exists := p.profiles[section.Name()]
				if exists {
					clio.Debugf("skipping profile with name %s - profile already defined in config", section.Name())
					continue
				}
				p.ProfileNames = append(p.ProfileNames, section.Name())
				p.profiles[section.Name()] = &Profile{RawConfig: section, Name: section.Name(), File: loader.Path()}
			}
		}
	}
	return nil
}

// Helper function which returns true if provided profile name string does not contain illegal characters
func IsLegalProfileName(name string) bool {
	illegalProfileNameCharacters := regexp.MustCompile(`[\\[\];'" ]`)
	illegalChars := `\][;'"` // These characters break the config file format and are not allowed for profile names
	if illegalProfileNameCharacters.MatchString(name) {
		clio.Warnf("The profile %s cannot be loaded because the name contains one or more of these characters '%s'", name, illegalChars)
		clio.Infof("Try renaming the profile to '%s'", illegalProfileNameCharacters.ReplaceAllString(name, "-"))
		return false
	}
	return true
}

// InitialiseProfilesTree will initialise all profiles
// this means that the profile parent relations are walked and the profile type is determined
// use this if you need to know the type of every profile in the config
// for large configuations, this may be expensive
func (p *Profiles) InitialiseProfilesTree(ctx context.Context) {
	for _, v := range p.profiles {
		_ = v.init(ctx, p, 0)
	}
}

// LoadInitialisedProfile returns an initialised profile by name
// this means that all the parents have been loaded and the profile type is defined
func (p *Profiles) LoadInitialisedProfile(ctx context.Context, profile string) (*Profile, error) {
	pr, err := p.Profile(profile)
	if err != nil {
		return nil, err
	}

	// For config that has 'granted' prefix we need to convert this to AWS config fields
	// aws configuration
	if hasGrantedSSOPrefix(pr.RawConfig) {
		awsConfig, err := ParseGrantedSSOProfile(ctx, pr)
		if err != nil {
			return nil, err
		}
		pr.AWSConfig = *awsConfig
		pr.Initialised = true
		pr.ProfileType = "AWS_SSO"
		return pr, nil
	} else {
		for _, v := range pr.RawConfig.Keys() {
			if v.Name() == "credential_process" && strings.HasPrefix(v.Value(), build.GrantedBinaryName()) {
				awsConfig, err := config.LoadSharedConfigProfile(ctx, pr.Name, func(lsco *config.LoadSharedConfigOptions) { lsco.ConfigFiles = []string{pr.File} })
				if err != nil {
					return nil, err
				}

				pr.AWSConfig = awsConfig
				pr.AWSConfig.CredentialProcess = ""
				pr.Initialised = true
				pr.ProfileType = "AWS_IAM"
				pr.HasSecureStorageIAMCredentials = true
				return pr, nil
			}
		}
	}

	// default initializaton flow
	err = pr.init(ctx, p, 0)
	if err != nil {
		return nil, err
	}
	return pr, nil
}

// Initialize profile's AWS config by fetching credentials from plain-text-SSO-token
// located at default cache directory.
func (p *Profile) InitWithPlainTextSSOToken(ctx context.Context, awsCred aws.Credentials) error {
	p.Initialised = true
	p.ProfileType = "AWS_SSO"

	cfg, err := config.LoadSharedConfigProfile(ctx, p.Name, func(lsco *config.LoadSharedConfigOptions) { lsco.ConfigFiles = []string{p.File} })
	if err != nil {
		return err
	}

	p.AWSConfig = cfg
	p.AWSConfig.Credentials.SessionToken = awsCred.SessionToken
	p.AWSConfig.Credentials.AccessKeyID = awsCred.AccessKeyID
	p.AWSConfig.Credentials.SecretAccessKey = awsCred.SecretAccessKey
	p.AWSConfig.Credentials.Expires = awsCred.Expires

	return nil
}

// Make sure credentials are available and valid.
func (p *Profile) LoadPlainTextSSOToken(ctx context.Context, profile string) (aws.Credentials, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithSharedConfigProfile(profile),
	)
	if err != nil {
		return aws.Credentials{}, err
	}

	// Will return err if there is no SSO session or it has expired.
	// So, returning empty aws.Credentials instead of err here.
	awsConfig, err := cfg.Credentials.Retrieve((ctx))
	if err != nil {
		// If no cache file is not found then.
		if errors.Is(err, syscall.ENOENT) {
			return aws.Credentials{}, nil
		}

		// if the token has expired or invalid then
		if _, ok := err.(*ssocreds.InvalidTokenError); ok {
			return aws.Credentials{}, nil
		}

		return aws.Credentials{}, err
	}

	return awsConfig, nil
}

func (p *Profile) init(ctx context.Context, profiles *Profiles, depth int) error {
	if !p.Initialised {
		// Ensures this recursive call does not exceed a maximum depth
		// potentially triggered by bad config files with cycles in source_profile
		// In simple cases this seems to be picked up by the AWS sdk before the profiles are initialised which would log a debug message instead
		p.Initialised = true

		cfg, err := config.LoadSharedConfigProfile(ctx, p.Name, func(lsco *config.LoadSharedConfigOptions) { lsco.ConfigFiles = []string{p.File} })
		if err != nil {
			return err
		}

		// set the sso session if it exists and not overridden on the profile
		if p.SSOSession != nil {
			if cfg.SSOStartURL == "" {
				cfg.SSOStartURL = p.SSOSession.SSOStartURL
			}
			if cfg.SSORegion == "" {
				cfg.SSORegion = p.SSOSession.SSORegion
			}
		}
		p.AWSConfig = cfg

		if depth < 10 {
			if p.AWSConfig.SourceProfileName == "" {
				as := assumers
				for _, a := range as {
					if a.ProfileMatchesType(p.RawConfig, p.AWSConfig) {
						p.ProfileType = a.Type()
						break
					}
				}
			} else {
				sourceProfile, ok := profiles.profiles[p.AWSConfig.SourceProfileName]
				if ok {
					p.LoadingError = sourceProfile.init(ctx, profiles, depth+1)
					if p.LoadingError != nil {
						return p.LoadingError
					}
					p.ProfileType = sourceProfile.ProfileType
					p.Parents = append(sourceProfile.Parents, sourceProfile)
				} else {
					p.LoadingError = fmt.Errorf("failed to load a source-profile for profile: %s . You should fix the issue with the source profile before you can assume this profile.", p.Name)
					return p.LoadingError
				}
			}
		} else {
			p.LoadingError = fmt.Errorf("maximum source profile depth exceeded for profile %s\nthis indicates that you have a cyclic reference in your aws profiles.[profile dev]\nregion = ap-southeast-2\nsource_profile = prod\n\n[profile prod]\nregion = ap-southeast-2\nsource_profile = dev", p.Name)
			return p.LoadingError
		}
	}
	return nil
}

// Region will attempt to load the region on this profile, if it is not set,
// attempt to load the parent if it exists
// else attempts to use the sso-region
// else attempts to load the default config
// returns a region, and bool = true if the default region was used
func (p *Profile) Region(ctx context.Context) (string, error) {
	if !p.Initialised {
		return "", ErrProfileNotInitialised
	}

	if p.AWSConfig.Region != "" {
		return p.AWSConfig.Region, nil
	}
	if len(p.Parents) > 0 {
		// return the region of the direct parent
		return p.Parents[len(p.Parents)-1].Region(ctx)
	}
	if p.AWSConfig.SSORegion != "" {
		return p.AWSConfig.SSORegion, nil
	}
	// if no region set, and no parent, and no sso region return the default region
	defaultCfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return "", err
	}
	region := defaultCfg.Region
	if region == "" {
		return "", fmt.Errorf("region not set on profile %s, could not load a default AWS_REGION. Either set a default region `aws configure set default.region us-west-2` or add a region to your profile", p.Name)
	}
	return region, nil
}

func (c *Profile) AssumeConsole(ctx context.Context, configOpts ConfigOpts) (aws.Credentials, error) {
	return AssumerFromType(c.ProfileType).AssumeConsole(ctx, c, configOpts)
}

func (c *Profile) AssumeTerminal(ctx context.Context, configOpts ConfigOpts) (aws.Credentials, error) {
	return AssumerFromType(c.ProfileType).AssumeTerminal(ctx, c, configOpts)
}

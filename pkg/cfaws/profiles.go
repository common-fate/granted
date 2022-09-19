package cfaws

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/ssocreds"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	"github.com/bigkevmcd/go-configparser"
	"github.com/common-fate/granted/pkg/debug"
	"github.com/fatih/color"
)

type ConfigOpts struct {
	Duration time.Duration
	Args     []string
}

type Profile struct {
	// allows access to the raw values from the file
	RawConfig configparser.Dict
	Name      string
	// the file that this profile is from
	File        string
	ProfileType string

	// ordered from root to direct parent profile
	Parents []*Profile
	// the original config, some values may be empty strings depending on the type or profile
	AWSConfig    config.SharedConfig
	Initialised  bool
	LoadingError error
	// set to true if aws temp credentails are fetched from ~/.aws/sso/cache dir.
	IsDefaultCredential bool
}

var ErrProfileNotInitialised error = errors.New("profile not initialised")

var ErrProfileNotFound error = errors.New("profile not found")

type Profiles struct {
	// alphabetically sorted after first load
	ProfileNames []string
	Profiles     map[string]*Profile
}

func (p *Profiles) HasProfile(profile string) bool {
	_, ok := p.Profiles[profile]
	return ok
}

func (p *Profiles) Profile(profile string) (*Profile, error) {
	if c, ok := p.Profiles[profile]; ok {
		return c, nil
	}
	return nil, ErrProfileNotFound
}

func LoadProfiles() (*Profiles, error) {
	p := Profiles{Profiles: make(map[string]*Profile)}
	err := p.loadDefaultConfigFile()
	if err != nil {
		return nil, err
	}
	err = p.loadDefaultCredentialsFile()
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
func (p *Profiles) loadDefaultConfigFile() error {
	configPath := config.DefaultSharedConfigFilename()
	configFile, err := configparser.NewConfigParserFromFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	// Itterate through the config sections
	for _, section := range configFile.Sections() {
		rawConfig, err := configFile.Items(section)

		if err != nil {
			fmt.Fprintf(color.Error, "failed to parse a profile from your AWS config: %s Due to the following error: %s\n", section, err)
			continue
		}
		// Check if the section is prefixed with 'profile ' and that the profile has a name
		if ((strings.HasPrefix(section, "profile ") && len(section) > 8) || section == "default") && IsLegalProfileName(section) {
			name := strings.TrimPrefix(section, "profile ")
			p.ProfileNames = append(p.ProfileNames, name)
			p.Profiles[name] = &Profile{RawConfig: rawConfig, Name: name, File: configPath}
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
func (p *Profiles) loadDefaultCredentialsFile() error {
	//fetch parsed credentials file
	credsPath := config.DefaultSharedCredentialsFilename()
	credsFile, err := configparser.NewConfigParserFromFile(credsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, section := range credsFile.Sections() {
		rawConfig, err := credsFile.Items(section)
		if err != nil {
			fmt.Fprintf(color.Error, "failed to parse a profile from your AWS credentials: %s Due to the following error: %s\n", section, err)
			continue
		}
		// We only care about the non default sections for the credentials file (no profile prefix either)
		if section != "default" && IsLegalProfileName(section) {
			// check for a duplicate profile in the map and skip if present (config file should take precedence)
			_, exists := p.Profiles[section]
			if exists {
				debug.Fprintf(debug.VerbosityDebug, color.Output, "skipping profile with name %s - profile already defined in config", section)
				continue
			}
			p.ProfileNames = append(p.ProfileNames, section)
			p.Profiles[section] = &Profile{RawConfig: rawConfig, Name: section, File: credsPath}
		}
	}
	return nil
}

// Helper function which returns true if provided profile name string does not contain illegal characters
func IsLegalProfileName(name string) bool {
	illegalChars := "\\][;'\"" // These characters break the config file format and should not be usable for profile names
	if strings.ContainsAny(name, illegalChars) {
		fmt.Fprintf(color.Error, "warning, profile: %s cannot be loaded because it contains one or more of: '%s' in the name, try replacing these with '-'\n", name, illegalChars)
		return false
	}
	return true
}

// InitialiseProfilesTree will initialise all profiles
// this means that the profile prarent relations are walked and the profile type is determined
// use this if you need to know the type of every profile in the config
// for large configuations, this may be expensive
func (p *Profiles) InitialiseProfilesTree(ctx context.Context) {
	for _, v := range p.Profiles {
		_ = v.init(ctx, p, 0)
	}
}

type grantedSSOCreds struct {
	granted_sso_start_url    string
	granted_sso_region       string
	granted_sso_account_name string
	granted_sso_account_id   string
	granted_sso_role_name    string
	region                   string
}

func convertGrantedCredToAWSConfig(ctx context.Context, p *Profile, gConfig grantedSSOCreds) (*config.SharedConfig, error) {
	cfg, err := config.LoadSharedConfigProfile(ctx, p.Name, func(lsco *config.LoadSharedConfigOptions) { lsco.ConfigFiles = []string{p.File} })
	if err != nil {
		return nil, err
	}

	cfg.SSOAccountID = gConfig.granted_sso_account_id
	cfg.SSORegion = gConfig.granted_sso_region
	cfg.SSORoleName = gConfig.granted_sso_role_name
	cfg.SSOStartURL = gConfig.granted_sso_start_url

	return &cfg, nil
}

// LoadInitialisedProfile returns an initialised profile by name
// this means that all the parents have been loaded and the profile type is defined
func (p *Profiles) LoadInitialisedProfile(ctx context.Context, profile string) (*Profile, error) {
	pr, err := p.Profile(profile)
	if err != nil {
		return nil, err
	}

	// Check if the aws config file has granted prefix.
	if err = pr.IsValidGrantedProfile(); err == nil {

		grantedSSOCreds := grantedSSOCreds{
			granted_sso_start_url:    pr.RawConfig["granted_sso_start_url"],
			granted_sso_region:       pr.RawConfig["granted_sso_region"],
			granted_sso_account_name: pr.RawConfig["granted_sso_account_name"],
			granted_sso_account_id:   pr.RawConfig["granted_sso_account_id"],
			granted_sso_role_name:    pr.RawConfig["granted_sso_role_name"],
			region:                   pr.RawConfig["region"],
		}

		awsConfig, err := convertGrantedCredToAWSConfig(ctx, pr, grantedSSOCreds)
		if err != nil {
			return nil, err
		}

		pr.AWSConfig = *awsConfig

		cfg := aws.NewConfig()
		cfg.Region = awsConfig.SSORegion
		client := sso.NewFromConfig(*cfg)
		provider := ssocreds.New(client, awsConfig.SSOAccountID, awsConfig.SSORoleName, awsConfig.SSOStartURL)

		credentials, err := provider.Retrieve(ctx)
		if err != nil {
			// If no cache file is not found then.
			if errors.Is(err, syscall.ENOENT) {
				ctx = context.WithValue(ctx, "shouldSilentStdout", true)
				token, err := SSODeviceCodeFlow(ctx, *cfg, pr)
				if err != nil {
					return nil, err
				}

				ssoToken := CreatePlainTextSSO(*awsConfig, token)

				if err := ssoToken.DumpToCacheDirectory(); err != nil {
					return nil, err
				}

				// recursively call the same func
				// second run will have the necessary plain text sso token so this won't be called again.
				return p.LoadInitialisedProfile(ctx, pr.Name)
			}

			// if the token has expired or invalid then
			if _, ok := err.(*ssocreds.InvalidTokenError); ok {
				// TODO: Refactor passing this flag
				ctx = context.WithValue(ctx, "shouldSilentStdout", true)
				token, err := SSODeviceCodeFlow(ctx, *cfg, pr)
				if err != nil {
					return nil, err
				}

				ssoToken := CreatePlainTextSSO(*awsConfig, token)

				if err := ssoToken.DumpToCacheDirectory(); err != nil {
					return nil, err
				}

				// recursively call the same func
				// second run will have the necessary plain text sso token so this won't be called again.
				return p.LoadInitialisedProfile(ctx, pr.Name)
			}

			return nil, err
		}

		if credentials.HasKeys() {
			err = pr.InitWithDefaultConfig(ctx, p, credentials)
			if err != nil {
				return nil, err
			}

			pr.IsDefaultCredential = true
			return pr, nil
		}
	}

	// check if we have valid credentials in `~/.aws/sso/cache`
	awsCredentials, err := pr.LoadDefaultSSOConfig(ctx, pr.Name)
	if err != nil {
		return nil, err
	}

	if awsCredentials.HasKeys() {
		err = pr.InitWithDefaultConfig(ctx, p, awsCredentials)
		if err != nil {
			return nil, err
		}

		pr.IsDefaultCredential = true
		return pr, nil
	}

	err = pr.init(ctx, p, 0)
	if err != nil {
		return nil, err
	}
	pr.IsDefaultCredential = false
	return pr, nil
}

func (p *Profile) InitWithDefaultConfig(ctx context.Context, profiles *Profiles, awsCred aws.Credentials) error {
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

// For `granted login` cmd, we have to make sure 'granted' prefix
// is added to the aws config file.
func (p *Profile) IsValidGrantedProfile() error {
	requiredGrantedCredentials := []string{"granted_sso_start_url", "granted_sso_region", "granted_sso_account_id", "granted_sso_role_name", "region"}

	for _, value := range requiredGrantedCredentials {
		if _, ok := p.RawConfig[value]; !ok {
			return fmt.Errorf("Invalid aws config for granted login. %s is undefined but necessary \n", value)
		}
	}

	return nil
}

// Make sure credentials are available and valid.
func (p *Profile) LoadDefaultSSOConfig(ctx context.Context, profile string) (aws.Credentials, error) {
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
				sourceProfile, ok := profiles.Profiles[p.AWSConfig.SourceProfileName]
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

// Region will attempt to load the reason on this profile, if it is not set,
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

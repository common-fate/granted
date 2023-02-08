package granted

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	"github.com/common-fate/awsconfigfile"
	"github.com/common-fate/clio"
	"github.com/common-fate/clio/clierr"
	"github.com/common-fate/granted/pkg/cfaws"
	"github.com/common-fate/granted/pkg/securestorage"
	"github.com/urfave/cli/v2"
	"gopkg.in/ini.v1"
)

const profileSectionIllegalChars = ` \][;'"`

// regular expression that matches on the characters \][;'" including whitespace, but does not match anything between {{ }} so it does not check inside go templates
// this regex is used as a basic safeguard to help users avoid mistakes in their templates
// for example "{{ .AccountName }} {{ .RoleName }}" this is invalid because it has a whitespace separating the template elements
var profileSectionIllegalCharsRegex = regexp.MustCompile(`(?s)((?:^|[^\{])[\s\][;'"]|[\][;'"][\s]*(?:$|[^\}]))`)
var matchGoTemplateSection = regexp.MustCompile(`\{\{[\s\S]*?\}\}`)

var defaultProfileNameTemplate = "{{ .AccountName }}/{{ .RoleName }}"

var SSOCommand = cli.Command{
	Name:        "sso",
	Usage:       "Manage your local AWS configuration file from information available in AWS SSO",
	Subcommands: []*cli.Command{&GenerateCommand, &PopulateCommand},
}

var GenerateCommand = cli.Command{
	Name:      "generate",
	Usage:     "Prints an AWS configuration file to stdout with profiles from accounts and roles available in AWS SSO",
	UsageText: "granted [global options] sso generate [command options] [sso-start-url]",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "prefix", Usage: "Specify a prefix for all generated profile names"},
		&cli.StringFlag{Name: "sso-region", Usage: "Specify the SSO region"},
		&cli.StringFlag{Name: "region", Usage: "The SSO region. Deprecated, use --sso-region instead. In future, this will be the AWS region for the profile to use", DefaultText: "us-east-1"},
		&cli.BoolFlag{Name: "no-credential-process", Usage: "Generate profiles without the Granted credential-process integration"},
		&cli.StringFlag{Name: "profile-template", Usage: "Specify profile name template", Value: defaultProfileNameTemplate}},
	Action: func(c *cli.Context) error {
		startURL := c.Args().First()
		if startURL == "" {
			return clierr.New("Usage: granted sso generate [sso-start-url]", clierr.Info("For example, granted sso generate https://example.awsapps.com/start"))
		}
		// if neither --sso-region or --region were set, show a warning to the user as we plan to make --sso-region required in future
		if !c.IsSet("region") && !c.IsSet("sso-region") {
			clio.Warnf("Please specify the --sso-region flag: 'granted sso populate --sso-region us-east-1 %s'", startURL)
			clio.Warn("Currently, Granted defaults to using us-east-1 if this is not provided. In a future version, this flag will be required (https://github.com/common-fate/granted/issues/271)")
		}

		options, err := parseCliOptions(c)
		if err != nil {
			return err
		}

		profiles, err := listSSOProfiles(c.Context, ListSSOProfilesInput{
			StartUrl:  startURL,
			SSORegion: options.SSORegion,
		})
		if err != nil {
			return err
		}

		config := ini.Empty()
		err = awsconfigfile.Merge(awsconfigfile.MergeOpts{
			Config:              config,
			SectionNameTemplate: options.ProfileTemplate,
			Profiles:            profiles,
			NoCredentialProcess: c.Bool("no-credential-process"),
		})
		if err != nil {
			return err
		}

		_, err = config.WriteTo(os.Stdout)
		return err
	},
}

var PopulateCommand = cli.Command{
	Name:      "populate",
	Usage:     "Populate your local AWS configuration file with profiles from accounts and roles available in AWS SSO",
	UsageText: "granted [global options] sso populate [command options] [sso-start-url]",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "prefix", Usage: "Specify a prefix for all generated profile names"},
		&cli.StringFlag{Name: "sso-region", Usage: "Specify the SSO region"},
		&cli.StringFlag{Name: "region", Usage: "The SSO region. Deprecated, use --sso-region instead. In future, this will be the AWS region for the profile to use", DefaultText: "us-east-1"}, &cli.BoolFlag{Name: "no-credential-process", Usage: "Generate profiles without the Granted credential-process integration"},
		&cli.StringFlag{Name: "profile-template", Usage: "Specify profile name template", Value: defaultProfileNameTemplate}},
	Action: func(c *cli.Context) error {
		startURL := c.Args().First()
		if startURL == "" {
			return clierr.New("Usage: granted sso populate [sso-start-url]", clierr.Info("For example, granted sso populate https://example.awsapps.com/start"))
		}

		// if neither --sso-region or --region were set, show a warning to the user as we plan to make --sso-region required in future
		if !c.IsSet("region") && !c.IsSet("sso-region") {
			clio.Warnf("Please specify the --sso-region flag: 'granted sso populate --sso-region us-east-1 %s'", startURL)
			clio.Warn("Currently, Granted defaults to using us-east-1 if this is not provided. In a future version, this flag will be required (https://github.com/common-fate/granted/issues/271)")
		}

		options, err := parseCliOptions(c)
		if err != nil {
			return err
		}

		profiles, err := listSSOProfiles(c.Context, ListSSOProfilesInput{
			StartUrl:  startURL,
			SSORegion: options.SSORegion,
		})
		if err != nil {
			return err
		}

		configFilename := config.DefaultSharedConfigFilename()

		config, err := ini.LoadSources(ini.LoadOptions{
			AllowNonUniqueSections:  false,
			SkipUnrecognizableLines: false,
		}, configFilename)
		if err != nil {
			if !os.IsNotExist(err) {
				return err
			}
			config = ini.Empty()
		}

		err = awsconfigfile.Merge(awsconfigfile.MergeOpts{
			Config:              config,
			SectionNameTemplate: options.ProfileTemplate,
			Profiles:            profiles,
			NoCredentialProcess: c.Bool("no-credential-process"),
		})
		if err != nil {
			return err
		}

		err = config.SaveTo(configFilename)
		if err != nil {
			return err
		}
		return nil
	},
}

func parseCliOptions(c *cli.Context) (*SSOCommonOptions, error) {
	prefix := c.String("prefix")

	if c.IsSet("region") {
		clio.Warn("Please use --sso-region rather than --region.")
		clio.Warn("In a future version of Granted, the --region flag will be used to specify the 'region' field in generated profiles, rather than the 'sso_region' field. (https://github.com/common-fate/granted/issues/271)")
	}

	// try --sso-region first, then fall back to --region.
	region := c.String("sso-region")
	if region == "" {
		region = c.String("region")
	}

	profileTemplate := c.String("profile-template")

	if strings.ContainsAny(prefix, profileSectionIllegalChars) {
		return nil, fmt.Errorf("--prefix flag must not contains illegal characters (%s)", profileSectionIllegalChars)
	}

	ssoRegion, err := cfaws.ExpandRegion(region)
	if err != nil {
		return nil, err
	}

	// check the profile template for any invalid section name characters
	if profileTemplate != defaultProfileNameTemplate {
		cleaned := matchGoTemplateSection.ReplaceAllString(profileTemplate, "")
		if profileSectionIllegalCharsRegex.MatchString(cleaned) {
			return nil, fmt.Errorf("--profile-template flag must not contain any of these illegal characters (%s)", profileSectionIllegalChars)
		}
	}

	options := SSOCommonOptions{
		Prefix:          prefix,
		SSORegion:       ssoRegion,
		ProfileTemplate: profileTemplate,
	}

	return &options, nil
}

type SSOCommonOptions struct {
	Prefix          string
	SSORegion       string
	ProfileTemplate string
}

type ListSSOProfilesInput struct {
	SSORegion string
	StartUrl  string
}

func listSSOProfiles(ctx context.Context, input ListSSOProfilesInput) ([]awsconfigfile.SSOProfile, error) {
	cfg := aws.NewConfig()
	cfg.Region = input.SSORegion
	secureSSOTokenStorage := securestorage.NewSecureSSOTokenStorage()
	ssoToken := secureSSOTokenStorage.GetValidSSOToken(input.StartUrl)
	var err error
	if ssoToken == nil {
		ssoToken, err = cfaws.SSODeviceCodeFlowFromStartUrl(ctx, *cfg, input.StartUrl)
		if err != nil {
			return nil, err
		}
	}

	clio.Info("fetching available profiles from AWS IAM Identity Center...")

	ssoClient := sso.NewFromConfig(*cfg)

	var ssoProfiles []awsconfigfile.SSOProfile

	listAccountsNextToken := ""
	for {
		listAccountsInput := sso.ListAccountsInput{AccessToken: &ssoToken.AccessToken}
		if listAccountsNextToken != "" {
			listAccountsInput.NextToken = &listAccountsNextToken
		}

		listAccountsOutput, err := ssoClient.ListAccounts(ctx, &listAccountsInput)
		if err != nil {
			return nil, err
		}

		for _, account := range listAccountsOutput.AccountList {
			listAccountRolesNextToken := ""
			for {
				listAccountRolesInput := sso.ListAccountRolesInput{
					AccessToken: &ssoToken.AccessToken,
					AccountId:   account.AccountId,
				}
				if listAccountRolesNextToken != "" {
					listAccountRolesInput.NextToken = &listAccountRolesNextToken
				}

				listAccountRolesOutput, err := ssoClient.ListAccountRoles(ctx, &listAccountRolesInput)
				if err != nil {
					return nil, err
				}

				for _, role := range listAccountRolesOutput.RoleList {
					ssoProfiles = append(ssoProfiles, awsconfigfile.SSOProfile{
						SSOStartURL:   input.StartUrl,
						SSORegion:     input.SSORegion,
						AccountID:     *role.AccountId,
						AccountName:   *account.AccountName,
						RoleName:      *role.RoleName,
						GeneratedFrom: "aws-sso",
					})
				}

				if listAccountRolesOutput.NextToken == nil {
					break
				}

				listAccountRolesNextToken = *listAccountRolesOutput.NextToken
			}
		}

		if listAccountsOutput.NextToken == nil {
			break
		}

		listAccountsNextToken = *listAccountsOutput.NextToken
	}

	return ssoProfiles, nil
}

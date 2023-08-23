package granted

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"net/http"
	"os"
	"regexp"

	"github.com/AlecAivazis/survey/v2"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	"github.com/common-fate/awsconfigfile"
	"github.com/common-fate/cli/cmd/command"
	"github.com/common-fate/cli/pkg/client"
	cfconfig "github.com/common-fate/cli/pkg/config"
	"github.com/common-fate/cli/pkg/profilesource"
	"github.com/common-fate/clio"
	"github.com/common-fate/clio/clierr"
	"github.com/common-fate/granted/pkg/cfaws"
	"github.com/common-fate/granted/pkg/securestorage"
	"github.com/common-fate/granted/pkg/testable"
	"github.com/urfave/cli/v2"
	"gopkg.in/ini.v1"
)

var SSOCommand = cli.Command{
	Name:        "sso",
	Usage:       "Manage your local AWS configuration file from information available in AWS SSO",
	Subcommands: []*cli.Command{&GenerateCommand, &PopulateCommand, &LoginCommand},
}

// in dev:
// go run ./cmd/granted/main.go sso generate --sso-region ap-southeast-2 [url]
var GenerateCommand = cli.Command{
	Name:      "generate",
	Usage:     "Prints an AWS configuration file to stdout with profiles from accounts and roles available in AWS SSO",
	UsageText: "granted [global options] sso generate [command options] [sso-start-url]",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "prefix", Usage: "Specify a prefix for all generated profile names"},
		&cli.StringFlag{Name: "sso-region", Usage: "Specify the SSO region"},
		&cli.StringSliceFlag{Name: "source", Usage: "The sources to load AWS profiles from (valid values are: 'aws-sso', 'commonfate')", Value: cli.NewStringSlice("aws-sso")},
		&cli.StringFlag{Name: "region", Usage: "The SSO region. Deprecated, use --sso-region instead. In future, this will be the AWS region for the profile to use", DefaultText: "us-east-1"},
		&cli.BoolFlag{Name: "no-credential-process", Usage: "Generate profiles without the Granted credential-process integration"},
		&cli.StringFlag{Name: "profile-template", Usage: "Specify profile name template", Value: awsconfigfile.DefaultProfileNameTemplate}},
	Action: func(c *cli.Context) error {
		ctx := c.Context
		fullCommand := fmt.Sprintf("%s %s", c.App.Name, c.Command.FullName()) // e.g. 'granted sso populate'

		startURL := c.Args().First()
		if startURL == "" {
			return clierr.New(fmt.Sprintf("Usage: %s [sso-start-url]", fullCommand), clierr.Infof("For example, %s https://example.awsapps.com/start", fullCommand))
		}

		// the --region flag behaviour will change in future: https://github.com/common-fate/granted/issues/360
		//
		// if neither --sso-region or --region were set, show a warning to the user as we plan to make --sso-region required in future
		if !c.IsSet("region") && !c.IsSet("sso-region") {
			clio.Warnf("Please specify the --sso-region flag: '%s --sso-region us-east-1 %s'", fullCommand, startURL)
			clio.Warn("Currently, Granted defaults to using us-east-1 if this is not provided. In a future version, this flag will be required (https://github.com/common-fate/granted/issues/360)")
		}

		if c.IsSet("region") {
			clio.Warn("Please use --sso-region rather than --region.")
			clio.Warn("In a future version of Granted, the --region flag will be used to specify the 'region' field in generated profiles, rather than the 'sso_region' field. (https://github.com/common-fate/granted/issues/360)")
		}

		// try --sso-region first, then fall back to --region.
		region := c.String("sso-region")
		if region == "" {
			region = c.String("region")
		}

		// end of --region flag behaviour warnings. These can be removed once https://github.com/common-fate/granted/issues/360 is closed.

		g := awsconfigfile.Generator{
			Config:              ini.Empty(),
			ProfileNameTemplate: c.String("profile-template"),
			NoCredentialProcess: c.Bool("no-credential-process"),
			Prefix:              c.String("prefix"),
		}

		for _, s := range c.StringSlice("source") {
			switch s {
			case "aws-sso":
				g.AddSource(AWSSSOSource{SSORegion: region, StartURL: startURL})
			case "commonfate", "common-fate", "cf":
				ps, err := getCFProfileSource(c, region, startURL)
				if err != nil {
					return err
				}
				g.AddSource(ps)
			default:
				return fmt.Errorf("unknown profile source %s: allowed sources are aws-sso, commonfate", s)
			}
		}

		err := g.Generate(ctx)
		if err != nil {
			return err
		}

		_, err = g.Config.WriteTo(os.Stdout)
		if err != nil {
			return err
		}

		return nil
	},
}

var PopulateCommand = cli.Command{
	Name:      "populate",
	Usage:     "Populate your local AWS configuration file with profiles from accounts and roles available in AWS SSO",
	UsageText: "granted [global options] sso populate [command options] [sso-start-url]",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "prefix", Usage: "Specify a prefix for all generated profile names"},
		&cli.StringFlag{Name: "sso-region", Usage: "Specify the SSO region"},
		&cli.StringSliceFlag{Name: "source", Usage: "The sources to load AWS profiles from", Value: cli.NewStringSlice("aws-sso")},
		&cli.BoolFlag{Name: "prune", Usage: "Remove any generated profiles with the 'common_fate_generated_from' key which no longer exist"},
		&cli.StringFlag{Name: "region", Usage: "The SSO region. Deprecated, use --sso-region instead. In future, this will be the AWS region for the profile to use", DefaultText: "us-east-1"}, &cli.BoolFlag{Name: "no-credential-process", Usage: "Generate profiles without the Granted credential-process integration"},
		&cli.StringFlag{Name: "profile-template", Usage: "Specify profile name template", Value: awsconfigfile.DefaultProfileNameTemplate}},
	Action: func(c *cli.Context) error {
		ctx := c.Context
		fullCommand := fmt.Sprintf("%s %s", c.App.Name, c.Command.FullName()) // e.g. 'granted sso populate'

		startURL := c.Args().First()
		if startURL == "" {
			return clierr.New(fmt.Sprintf("Usage: %s [sso-start-url]", fullCommand), clierr.Infof("For example, %s https://example.awsapps.com/start", fullCommand))
		}

		// the --region flag behaviour will change in future: https://github.com/common-fate/granted/issues/360
		//
		// if neither --sso-region or --region were set, show a warning to the user as we plan to make --sso-region required in future
		if !c.IsSet("region") && !c.IsSet("sso-region") {
			clio.Warnf("Please specify the --sso-region flag: '%s --sso-region us-east-1 %s'", fullCommand, startURL)
			clio.Warn("Currently, Granted defaults to using us-east-1 if this is not provided. In a future version, this flag will be required (https://github.com/common-fate/granted/issues/360)")
		}

		if c.IsSet("region") {
			clio.Warn("Please use --sso-region rather than --region.")
			clio.Warn("In a future version of Granted, the --region flag will be used to specify the 'region' field in generated profiles, rather than the 'sso_region' field. (https://github.com/common-fate/granted/issues/360)")
		}

		// try --sso-region first, then fall back to --region.
		region := c.String("sso-region")
		if region == "" {
			region = c.String("region")
		}

		// end of --region flag behaviour warnings. These can be removed once https://github.com/common-fate/granted/issues/360 is closed.
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

		var pruneStartURLs []string

		if c.Bool("prune") {
			pruneStartURLs = []string{startURL}
		}

		g := awsconfigfile.Generator{
			Config:              config,
			ProfileNameTemplate: c.String("profile-template"),
			NoCredentialProcess: c.Bool("no-credential-process"),
			Prefix:              c.String("prefix"),
			PruneStartURLs:      pruneStartURLs,
		}

		for _, s := range c.StringSlice("source") {
			switch s {
			case "aws-sso":
				g.AddSource(AWSSSOSource{SSORegion: region, StartURL: startURL})
			case "commonfate", "common-fate", "cf":
				ps, err := getCFProfileSource(c, region, startURL)
				if err != nil {
					return err
				}
				g.AddSource(ps)
			default:
				return fmt.Errorf("unknown profile source %s: allowed sources are aws-sso, commonfate", s)
			}
		}
		err = g.Generate(ctx)
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

var LoginCommand = cli.Command{
	Name:  "login",
	Usage: "Log in via AWS SSO interactive credential process",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "sso-region", Usage: "Specify the SSO region"},
		&cli.StringFlag{Name: "sso-start-url", Usage: "Specify the SSO start url"},
	},
	Action: func(c *cli.Context) error {
		ctx := c.Context
		ssoStartUrl := c.String("sso-start-url")

		if ssoStartUrl == "" {
			in1 := survey.Input{Message: "SSO Start URL"}
			err := testable.AskOne(&in1, &ssoStartUrl)
			if err != nil {
				return err
			}
		}

		ssoRegion := c.String("sso-region")

		if ssoRegion == "" {

			// fetch the start url to extract the region from the html
			resp, err := http.Get(ssoStartUrl)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			// extract the region using a regex on the meta tag "region"
			re := regexp.MustCompile(`<meta\s+name="region"\s+content="(.*?)"/>`)
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}

			match := re.FindStringSubmatch(string(body))
			if len(match) == 2 {
				ssoRegion = match[1]
			}

			// Fallback to user input
			if ssoRegion == "" {
				in2 := survey.Input{Message: "Region"}
				err := testable.AskOne(&in2, &ssoRegion)
				if err != nil {
					return err
				}
			}
		}

		cfg := aws.NewConfig()
		cfg.Region = ssoRegion

		secureSSOTokenStorage := securestorage.NewSecureSSOTokenStorage()

		newSSOToken, err := cfaws.SSODeviceCodeFlowFromStartUrl(ctx, *cfg, ssoStartUrl)
		if err != nil {
			return err
		}

		secureSSOTokenStorage.StoreSSOToken(ssoStartUrl, *newSSOToken)

		clio.Successf("Successfully logged into Start URL: %s", ssoStartUrl)

		return nil
	},
}

func getCFProfileSource(c *cli.Context, region, startURL string) (profilesource.Source, error) {
	kr, err := securestorage.NewCF().Storage.Keyring()
	if err != nil {
		return profilesource.Source{}, err
	}

	// login if the CF API isn't configured
	if !cfconfig.IsConfigured() {
		lf := command.LoginFlow{Keyring: kr, ForceInteractive: true}
		err = lf.LoginAction(c)
		if err != nil {
			return profilesource.Source{}, err
		}
	}

	cfg, err := cfconfig.Load()
	if err != nil {
		return profilesource.Source{}, err
	}

	cf, err := client.FromConfig(c.Context, cfg,
		client.WithKeyring(kr),
		client.WithLoginHint("granted login"),
	)
	if err != nil {
		return profilesource.Source{}, err
	}

	ps := profilesource.Source{SSORegion: region, StartURL: startURL, Client: cf, DashboardURL: cfg.CurrentOrEmpty().DashboardURL}

	clio.Infof("listing available profiles from Common Fate (%s)", ps.DashboardURL)

	return ps, nil
}

type AWSSSOSource struct {
	SSORegion string
	StartURL  string
}

func listAccountRoles(ctx context.Context, ssoClient sso.Client, listAccountRolesInput sso.ListAccountRolesInput, attemptCount int) (*sso.ListAccountRolesOutput, error) {
	listAccountRolesOutput, err := ssoClient.ListAccountRoles(ctx, &listAccountRolesInput)
	if err != nil {
		if strings.Contains(err.Error(), "StatusCode: 429") {
			if attemptCount > 3 {
				return nil, err
			}
			attemptCount += 1
			clio.Debugf("Encountered StatusCode 429...doing backoff...attempt count: %d", attemptCount)
			time.Sleep(time.Second * time.Duration(attemptCount))
			return listAccountRoles(ctx, ssoClient, listAccountRolesInput, attemptCount)
		}
		return nil, err
	}
	return listAccountRolesOutput, err
}

func (s AWSSSOSource) GetProfiles(ctx context.Context) ([]awsconfigfile.SSOProfile, error) {
	region, err := cfaws.ExpandRegion(s.SSORegion)
	if err != nil {
		return nil, err
	}

	cfg := aws.NewConfig()
	cfg.Region = region
	secureSSOTokenStorage := securestorage.NewSecureSSOTokenStorage()
	ssoTokenFromSecureCache := secureSSOTokenStorage.GetValidSSOToken(s.StartURL)
	ssoTokenFromPlainText := cfaws.GetValidSSOTokenFromPlaintextCache(s.StartURL)

	// depending on whether creds come from secure storage or ~/.aws/sso/cache, we need to use different access tokens
	accessToken := ""

	// we also want to store this in the secure cache to prevent subsequent logins
	if ssoTokenFromPlainText != nil {
		secureSSOTokenStorage.StoreSSOToken(s.StartURL, *ssoTokenFromPlainText)
	}

	if ssoTokenFromSecureCache == nil && ssoTokenFromPlainText == nil {
		// otherwise, login with SSO
		ssoTokenFromSecureCache, err = cfaws.SSODeviceCodeFlowFromStartUrl(ctx, *cfg, s.StartURL)
		if err != nil {
			return nil, err
		}
		secureSSOTokenStorage.StoreSSOToken(s.StartURL, *ssoTokenFromSecureCache)
	}

	if ssoTokenFromSecureCache != nil {
		accessToken = ssoTokenFromSecureCache.AccessToken
	} else {
		accessToken = ssoTokenFromPlainText.AccessToken
	}

	clio.Info("listing available profiles from AWS IAM Identity Center...")

	ssoClient := sso.NewFromConfig(*cfg)

	// if the token is nil fetch it from config instead
	var ssoProfiles []awsconfigfile.SSOProfile
	listAccountsNextToken := ""
	for {
		listAccountsInput := sso.ListAccountsInput{
			AccessToken: &accessToken,
		}
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
					AccessToken: &accessToken,
					AccountId:   account.AccountId,
				}
				if listAccountRolesNextToken != "" {
					listAccountRolesInput.NextToken = &listAccountRolesNextToken
				}

				listAccountRolesOutput, err := listAccountRoles(ctx, *ssoClient, listAccountRolesInput, 0)
				if err != nil {
					return nil, err
				}

				for _, role := range listAccountRolesOutput.RoleList {
					ssoProfiles = append(ssoProfiles, awsconfigfile.SSOProfile{
						SSOStartURL:   s.StartURL,
						SSORegion:     region,
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

package granted

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	"github.com/bigkevmcd/go-configparser"
	"github.com/common-fate/granted/pkg/cfaws"
	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
)

var SSOCommand = cli.Command{
	Name:        "sso",
	Usage:       "Manage AWS Config from information available in AWS SSO",
	Subcommands: []*cli.Command{&GenerateCommand, &PopulateCommand},
}

var GenerateCommand = cli.Command{
	Name:      "generate",
	Usage:     "Outputs an AWS Config with profiles from accounts and roles available in AWS SSO",
	UsageText: "granted [global options] sso generate [command options] [sso-start-url]",
	Flags:     []cli.Flag{&cli.StringFlag{Name: "prefix", Usage: "Specify a prefix for all generated profile names"}, &cli.StringFlag{Name: "region", Usage: "Specify the SSO region", DefaultText: "us-east-1"}},
	Action: func(c *cli.Context) error {
		options, err := parseCliOptions(c)
		if err != nil {
			return err
		}

		ssoProfiles, err := listSSOProfiles(c.Context, ListSSOProfilesInput{
			StartUrl:  options.StartUrl,
			SSORegion: options.SSORegion,
		})
		if err != nil {
			return err
		}

		config := configparser.New()

		err = mergeSSOProfiles(config, options.Prefix, ssoProfiles)
		if err != nil {
			return err
		}

		// configparser can't create a string so we can print the config
		// to the screen, so we work our way through the sections and
		// section items manually.
		for sectionIdx, sectionName := range config.Sections() {
			if sectionIdx != 0 {
				fmt.Fprintln(color.Output)
			}

			fmt.Fprintln(color.Output, "["+sectionName+"]")
			items, err := config.Items(sectionName)
			if err != nil {
				return nil
			}

			for key, value := range items {
				fmt.Fprintln(color.Output, key+" = "+value)
			}
		}

		return nil
	},
}

var PopulateCommand = cli.Command{
	Name:      "populate",
	Usage:     "Populate your AWS Config with profiles from accounts and roles available in AWS SSO",
	UsageText: "granted [global options] sso populate [command options] [sso-start-url]",
	Flags:     []cli.Flag{&cli.StringFlag{Name: "prefix", Usage: "Specify a prefix for all generated profile names"}, &cli.StringFlag{Name: "region", Usage: "Specify the SSO region", DefaultText: "us-east-1"}},
	Action: func(c *cli.Context) error {
		options, err := parseCliOptions(c)
		if err != nil {
			return err
		}

		ssoProfiles, err := listSSOProfiles(c.Context, ListSSOProfilesInput{
			StartUrl:  options.StartUrl,
			SSORegion: options.SSORegion,
		})
		if err != nil {
			return err
		}

		configFilename := config.DefaultSharedConfigFilename()

		// Use the existing config or create if it doesn't exist.
		config, err := configparser.NewConfigParserFromFile(configFilename)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return err
			}
			config = configparser.New()
		}

		if err := mergeSSOProfiles(config, options.Prefix, ssoProfiles); err != nil {
			return err
		}

		err = config.SaveWithDelimiter(configFilename, "=")
		if err != nil {
			return err
		}
		return nil
	},
}

func parseCliOptions(c *cli.Context) (*SSOCommonOptions, error) {
	prefix := c.String("prefix")
	match, err := regexp.MatchString("^[A-Za-z0-9_-]*$", prefix)
	if err != nil {
		return nil, err
	}

	if !match {
		return nil, fmt.Errorf("--prefix flag must be alpha-numeric, underscores or hyphens")
	}

	ssoRegion, err := cfaws.ExpandRegion(c.String("region"))
	if err != nil {
		return nil, err
	}

	if c.Args().Len() != 1 {
		return nil, fmt.Errorf("Please provide an sso start url")
	}

	startUrl := c.Args().First()

	options := SSOCommonOptions{
		Prefix:    prefix,
		StartUrl:  startUrl,
		SSORegion: ssoRegion,
	}

	return &options, nil
}

type SSOCommonOptions struct {
	Prefix    string
	StartUrl  string
	SSORegion string
}

type ListSSOProfilesInput struct {
	SSORegion string
	StartUrl  string
}

type SSOProfile struct {
	// SSO details
	StartUrl  string
	SSORegion string
	// Account and role details
	AccountId   string
	AccountName string
	RoleName    string
}

func listSSOProfiles(ctx context.Context, input ListSSOProfilesInput) ([]SSOProfile, error) {
	cfg := aws.NewConfig()
	cfg.Region = input.SSORegion

	ssoToken, err := cfaws.SSODeviceCodeFlowFromStartUrl(ctx, *cfg, input.StartUrl)
	if err != nil {
		return nil, err
	}

	ssoClient := sso.NewFromConfig(*cfg)

	var ssoProfiles []SSOProfile

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
					ssoProfiles = append(ssoProfiles, SSOProfile{
						StartUrl:    input.StartUrl,
						SSORegion:   input.SSORegion,
						AccountId:   *role.AccountId,
						AccountName: *account.AccountName,
						RoleName:    *role.RoleName,
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

func mergeSSOProfiles(config *configparser.ConfigParser, prefix string, ssoProfiles []SSOProfile) error {
	for _, ssoProfile := range ssoProfiles {
		sectionName := "profile " + prefix + normalizeAccountName(ssoProfile.AccountName) + "-" + ssoProfile.RoleName

		if config.HasSection(sectionName) {
			err := config.RemoveSection(sectionName)
			if err != nil {
				return err
			}
		}

		if err := config.AddSection(sectionName); err != nil {
			return err
		}

		err := config.Set(sectionName, "sso_start_url", ssoProfile.StartUrl)
		if err != nil {
			return err
		}
		err = config.Set(sectionName, "sso_region", ssoProfile.SSORegion)
		if err != nil {
			return err
		}
		err = config.Set(sectionName, "sso_account_id", ssoProfile.AccountId)
		if err != nil {
			return err
		}
		err = config.Set(sectionName, "sso_role_name", ssoProfile.RoleName)
		if err != nil {
			return err
		}
	}

	return nil
}

func normalizeAccountName(accountName string) string {
	return strings.ReplaceAll(accountName, " ", "-")
}

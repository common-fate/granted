package profilegen

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	"github.com/common-fate/awsconfigfile"
	"github.com/common-fate/clio"
	"github.com/common-fate/granted/pkg/cfaws"
	"github.com/common-fate/granted/pkg/securestorage"
)

type AWSSSOSource struct {
	SSORegion string
	StartURL  string
}

func (s AWSSSOSource) GetProfiles(ctx context.Context) ([]awsconfigfile.SSOProfile, error) {
	cfg := aws.NewConfig()
	cfg.Region = s.SSORegion
	secureSSOTokenStorage := securestorage.NewSecureSSOTokenStorage()
	ssoToken := secureSSOTokenStorage.GetValidSSOToken(s.StartURL)
	var err error
	if ssoToken == nil {
		ssoToken, err = cfaws.SSODeviceCodeFlowFromStartUrl(ctx, *cfg, s.StartURL)
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
						SSOStartURL:   s.StartURL,
						SSORegion:     s.SSORegion,
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

package assume

import (
	"bufio"
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
	"github.com/common-fate/granted/pkg/cfaws"
	"github.com/pkg/browser"
)

func GetSSO(ctx context.Context, config *cfaws.CFSharedConfig) {
	// IO = Determine Required Config Values to Establish SSO Session ✅
	// IO = Establish an SSO Session with config vars ✅
	// IO = Retreive any relevant Credentials from the SSO Session
	// IO = Export the credentials to the environment

	cfg, err := config.AwsConfig(ctx)
	if err != nil {
		fmt.Println(err)
	}

	if err != nil {
		fmt.Println(err)
	}
	ssooidcClient := ssooidc.NewFromConfig(cfg)
	if err != nil {
		fmt.Println(err)
	}
	register, err := ssooidcClient.RegisterClient(ctx, &ssooidc.RegisterClientInput{
		ClientName: aws.String("sample-client-name"),
		ClientType: aws.String("public"),
		Scopes:     []string{"sso-portal:*"},
	})
	if err != nil {
		fmt.Println(err)
	}

	// authorize your device using the client registration response
	deviceAuth, err := ssooidcClient.StartDeviceAuthorization(ctx, &ssooidc.StartDeviceAuthorizationInput{
		ClientId:     register.ClientId,
		ClientSecret: register.ClientSecret,
		StartUrl:     aws.String(config.RawConfig.SSOStartURL),
	})
	if err != nil {
		fmt.Println(err)
	}
	// trigger OIDC login. open browser to login. close tab once login is done. press enter to continue
	url := aws.ToString(deviceAuth.VerificationUriComplete)
	fmt.Printf("If browser is not opened automatically, please open link:\n%v\n", url)
	err = browser.OpenURL(url)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Press ENTER key once login is done")
	_ = bufio.NewScanner(os.Stdin).Scan()
	// generate sso token
	token, err := ssooidcClient.CreateToken(ctx, &ssooidc.CreateTokenInput{
		ClientId:     register.ClientId,
		ClientSecret: register.ClientSecret,
		DeviceCode:   deviceAuth.DeviceCode,
		GrantType:    aws.String("urn:ietf:params:oauth:grant-type:device_code"),
	})
	if err != nil {
		fmt.Println(err)
	}
	// create sso client
	ssoClient := sso.NewFromConfig(cfg)

	// This may be unnecessary, but it reveals the full list of accounts per ssoClient
	fmt.Println("Fetching list of all accounts for user")
	accountPaginator := sso.NewListAccountsPaginator(ssoClient, &sso.ListAccountsInput{
		AccessToken: token.AccessToken,
	})
	for accountPaginator.HasMorePages() {
		x, err := accountPaginator.NextPage(ctx)
		if err != nil {
			fmt.Println(err)
		}
		for _, y := range x.AccountList {
			fmt.Println("-------------------------------------------------------")
			fmt.Printf("\nAccount ID: %v\nName: %v\nEmail: %v\n", aws.ToString(y.AccountId), aws.ToString(y.AccountName), aws.ToString(y.EmailAddress))

			// list roles for a given account [ONLY provided for better example coverage]
			fmt.Printf("\n\nFetching roles of account %v for user\n", aws.ToString(y.AccountId))
			rolePaginator := sso.NewListAccountRolesPaginator(ssoClient, &sso.ListAccountRolesInput{
				AccessToken: token.AccessToken,
				AccountId:   y.AccountId,
			})
			for rolePaginator.HasMorePages() {
				z, err := rolePaginator.NextPage(ctx)
				if err != nil {
					fmt.Println(err)
				}
				for _, p := range z.RoleList {
					fmt.Printf("Account ID: %v Role Name: %v\n", aws.ToString(p.AccountId), aws.ToString(p.RoleName))
				}
			}

		}
	}
	fmt.Println("-------------------------------------------------------")

}

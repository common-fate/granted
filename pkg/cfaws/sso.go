package cfaws

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	ssotypes "github.com/aws/aws-sdk-go-v2/service/sso/types"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
	ssooidctypes "github.com/aws/aws-sdk-go-v2/service/ssooidc/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/aws-sdk-go-v2/service/sts/types"
	"github.com/pkg/browser"
)

func (c *CFSharedConfig) SSOLogin(ctx context.Context) (aws.Credentials, error) {
	if c.ProfileType != ProfileTypeSSO {
		return aws.Credentials{}, errors.New("cannot ssologin to non sso profile")
	}

	rootProfile := c
	requiresAssuming := false
	if len(c.Parents) > 0 {
		rootProfile = c.Parents[0]
		requiresAssuming = true
	}

	cfg, err := rootProfile.AwsConfig(ctx)
	if err != nil {
		return aws.Credentials{}, err
	}
	cachedToken, _ := CheckSSOTokenStore(rootProfile.Name)
	newToken := false
	if cachedToken == nil {
		newToken = true
		cachedToken, err = SSODeviceCodeFlow(ctx, cfg, rootProfile)
		if err != nil {
			return aws.Credentials{}, err
		}
	}
	if newToken {
		err = WriteSSOToken(rootProfile.Name, *cachedToken)
		// only write errors for caching if its in debug mode
		// Don't block assuming
		if os.Getenv("DEBUG") == "true" && err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
		}
	}
	// create sso client
	ssoClient := sso.NewFromConfig(cfg)
	res, err := ssoClient.GetRoleCredentials(ctx, &sso.GetRoleCredentialsInput{AccessToken: &cachedToken.AccessToken, AccountId: &rootProfile.RawConfig.SSOAccountID, RoleName: &rootProfile.RawConfig.SSORoleName})
	if err != nil {
		var unauthorised *ssotypes.UnauthorizedException
		if errors.As(err, &unauthorised) {
			// possible error with the access token we used, in this case we should clear our cached token and request a new one if the user tries again
			_ = ClearSSOToken(rootProfile.Name)
		} else {
			return aws.Credentials{}, err
		}
	}

	rootCreds := TypeRoleCredsToAwsCreds(*res.RoleCredentials)
	credProvider := &CredProv{rootCreds}
	if requiresAssuming {

		toAssume := append([]*CFSharedConfig{}, c.Parents[1:]...)
		toAssume = append(toAssume, c)
		for i, p := range toAssume {
			stsClient := sts.New(sts.Options{Credentials: aws.NewCredentialsCache(credProvider), Region: p.RawConfig.Region})
			stsRes, err := stsClient.AssumeRole(ctx, &sts.AssumeRoleInput{
				RoleArn:         &p.RawConfig.RoleARN,
				RoleSessionName: &p.Name,
			})
			if err != nil {
				return aws.Credentials{}, err
			}
			// only print for sub assumes because the final credentials are printed at the end of the assume command
			// this is here for visibility in to role traversals when assuming a final profile with sso
			if i < len(toAssume)-1 {
				fmt.Fprintf(os.Stderr, "\033[32m\nAssumed parent profile: [%s] session credentials will expire %s\033[0m\n", p.Name, stsRes.Credentials.Expiration.Local().String())
			}
			credProvider = &CredProv{TypeCredsToAwsCreds(*stsRes.Credentials)}

		}
	}
	return credProvider.Credentials, nil

}
func SSODeviceCodeFlow(ctx context.Context, cfg aws.Config, rootProfile *CFSharedConfig) (*SSOToken, error) {
	ssooidcClient := ssooidc.NewFromConfig(cfg)

	register, err := ssooidcClient.RegisterClient(ctx, &ssooidc.RegisterClientInput{
		ClientName: aws.String("granted-cli-client"),
		ClientType: aws.String("public"),
		Scopes:     []string{"sso-portal:*"},
	})
	if err != nil {
		return nil, err
	}

	// authorize your device using the client registration response
	deviceAuth, err := ssooidcClient.StartDeviceAuthorization(ctx, &ssooidc.StartDeviceAuthorizationInput{
		ClientId:     register.ClientId,
		ClientSecret: register.ClientSecret,
		StartUrl:     aws.String(rootProfile.RawConfig.SSOStartURL),
	})
	if err != nil {
		return nil, err
	}
	// trigger OIDC login. open browser to login. close tab once login is done. press enter to continue
	url := aws.ToString(deviceAuth.VerificationUriComplete)
	fmt.Fprintf(os.Stderr, "If browser is not opened automatically, please open link:\n%v\n", url)
	err = browser.OpenURL(url)
	if err != nil {
		return nil, err
	}

	fmt.Fprintln(os.Stderr, "\nAwaiting authentication in the browser...")
	token, err := PollToken(ctx, ssooidcClient, *register.ClientSecret, *register.ClientId, *deviceAuth.DeviceCode, PollingConfig{CheckInterval: time.Second * 2, TimeoutAfter: time.Minute * 2})
	if err != nil {
		return nil, err
	}

	return &SSOToken{AccessToken: *token.AccessToken, Expiry: time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)}, nil

}

func TypeCredsToAwsCreds(c types.Credentials) aws.Credentials {
	return aws.Credentials{AccessKeyID: *c.AccessKeyId, SecretAccessKey: *c.SecretAccessKey, SessionToken: *c.SessionToken, CanExpire: true, Expires: *c.Expiration}
}
func TypeRoleCredsToAwsCreds(c ssotypes.RoleCredentials) aws.Credentials {
	return aws.Credentials{AccessKeyID: *c.AccessKeyId, SecretAccessKey: *c.SecretAccessKey, SessionToken: *c.SessionToken, CanExpire: true, Expires: time.UnixMilli(c.Expiration)}
}

type CredProv struct{ aws.Credentials }

func (c *CredProv) Retrieve(ctx context.Context) (aws.Credentials, error) {
	return c.Credentials, nil
}

var ErrTimeout error = errors.New("polling for device authorization token timed out")

type PollingConfig struct {
	CheckInterval time.Duration
	TimeoutAfter  time.Duration
}

//PollToken will poll for a token and return it once the authentication/authorization flow has been completed in the browser
func PollToken(ctx context.Context, c *ssooidc.Client, clientSecret string, clientID string, deviceCode string, cfg PollingConfig) (*ssooidc.CreateTokenOutput, error) {
	start := time.Now()
	for {
		time.Sleep(cfg.CheckInterval)

		token, err := c.CreateToken(ctx, &ssooidc.CreateTokenInput{
			ClientId:     &clientID,
			ClientSecret: &clientSecret,
			DeviceCode:   &deviceCode,
			GrantType:    aws.String("urn:ietf:params:oauth:grant-type:device_code"),
		})
		var pendingAuth *ssooidctypes.AuthorizationPendingException
		if err == nil {
			return token, nil
		} else if !errors.As(err, &pendingAuth) {
			return nil, err
		}

		if time.Now().After(start.Add(cfg.TimeoutAfter)) {
			return nil, ErrTimeout
		}
	}
}

// // This may be unnecessary, but it reveals the full list of accounts per ssoClient
// fmt.Println("Fetching list of all accounts for user")
// accountPaginator := sso.NewListAccountsPaginator(ssoClient, &sso.ListAccountsInput{
// 	AccessToken: token.AccessToken,
// })
// for accountPaginator.HasMorePages() {
// 	x, err := accountPaginator.NextPage(ctx)
// 	if err != nil {
// 		return err
// 	}
// 	for _, y := range x.AccountList {
// 		fmt.Println("-------------------------------------------------------")
// 		fmt.Printf("\nAccount ID: %v\nName: %v\nEmail: %v\n", aws.ToString(y.AccountId), aws.ToString(y.AccountName), aws.ToString(y.EmailAddress))

// 		// list roles for a given account [ONLY provided for better example coverage]
// 		fmt.Printf("\n\nFetching roles of account %v for user\n", aws.ToString(y.AccountId))
// 		rolePaginator := sso.NewListAccountRolesPaginator(ssoClient, &sso.ListAccountRolesInput{
// 			AccessToken: token.AccessToken,
// 			AccountId:   y.AccountId,
// 		})
// 		for rolePaginator.HasMorePages() {
// 			z, err := rolePaginator.NextPage(ctx)
// 			if err != nil {
// 				return err
// 			}
// 			for _, p := range z.RoleList {
// 				fmt.Printf("Account ID: %v Role Name: %v\n", aws.ToString(p.AccountId), aws.ToString(p.RoleName))
// 			}
// 		}

// 	}
// }
// fmt.Println("-------------------------------------------------------")

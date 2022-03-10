package cfaws

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	ssotypes "github.com/aws/aws-sdk-go-v2/service/sso/types"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
	ssooidctypes "github.com/aws/aws-sdk-go-v2/service/ssooidc/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/bigkevmcd/go-configparser"
	"github.com/pkg/browser"
	"github.com/pkg/errors"
)

// Implements Assumer
type AwsSsoAssumer struct {
}

func (asa *AwsSsoAssumer) AssumeTerminal(ctx context.Context, c *CFSharedConfig, args []string) (aws.Credentials, error) {
	return c.SSOLogin(ctx)
}

func (asa *AwsSsoAssumer) AssumeConsole(ctx context.Context, c *CFSharedConfig, args []string) (aws.Credentials, error) {
	return c.SSOLogin(ctx)
}

func (asa *AwsSsoAssumer) Type() string {
	return "AWS_SSO"
}

// Matches the profile type on whether it is an sso profile by checking for ssoaccountid.
func (asa *AwsSsoAssumer) ProfileMatchesType(rawProfile configparser.Dict, parsedProfile config.SharedConfig) bool {
	return parsedProfile.SSOAccountID != ""
}

func (c *CFSharedConfig) SSOLogin(ctx context.Context) (aws.Credentials, error) {

	rootProfile := c
	requiresAssuming := false
	if len(c.Parents) > 0 {
		rootProfile = c.Parents[0]
		requiresAssuming = true
	}

	ssoTokenKey := rootProfile.AWSConfig.SSOStartURL
	cfg, err := rootProfile.AwsConfig(ctx, true)
	if err != nil {
		return aws.Credentials{}, err
	}
	cachedToken := GetValidCachedToken(ssoTokenKey)
	if cachedToken == nil {
		cachedToken, err = SSODeviceCodeFlow(ctx, cfg, rootProfile)
		if err != nil {
			return aws.Credentials{}, err
		}

	}

	StoreSSOToken(ssoTokenKey, *cachedToken, rootProfile.Name)

	// create sso client
	ssoClient := sso.NewFromConfig(cfg)
	res, err := ssoClient.GetRoleCredentials(ctx, &sso.GetRoleCredentialsInput{AccessToken: &cachedToken.AccessToken, AccountId: &rootProfile.AWSConfig.SSOAccountID, RoleName: &rootProfile.AWSConfig.SSORoleName})
	if err != nil {
		var unauthorised *ssotypes.UnauthorizedException
		if errors.As(err, &unauthorised) {
			// possible error with the access token we used, in this case we should clear our cached token and request a new one if the user tries again
			ClearSSOToken(ssoTokenKey)
		}
		return aws.Credentials{}, err

	}

	rootCreds := TypeRoleCredsToAwsCreds(*res.RoleCredentials)
	credProvider := &CredProv{rootCreds}

	if requiresAssuming {

		toAssume := append([]*CFSharedConfig{}, c.Parents[1:]...)
		toAssume = append(toAssume, c)
		for i, p := range toAssume {
			region, _, err := c.Region(ctx)
			if err != nil {
				return aws.Credentials{}, err
			}
			// in order to support profiles which do not specify a region, we use the default region when assuming the role

			stsClient := sts.New(sts.Options{Credentials: aws.NewCredentialsCache(credProvider), Region: region})
			stsRes, err := stsClient.AssumeRole(ctx, &sts.AssumeRoleInput{
				RoleArn:         &p.AWSConfig.RoleARN,
				RoleSessionName: &p.Name,
			})
			if err != nil {
				return aws.Credentials{}, err
			}
			// only print for sub assumes because the final credentials are printed at the end of the assume command
			// this is here for visibility in to role traversals when assuming a final profile with sso
			if i < len(toAssume)-1 {
				fmt.Fprintf(os.Stderr, "\033[32m\nAssumed parent profile: [%s](%s) session credentials will expire %s\033[0m\n", p.Name, region, stsRes.Credentials.Expiration.Local().String())
			}
			credProvider = &CredProv{TypeCredsToAwsCreds(*stsRes.Credentials)}

		}
	}
	return credProvider.Credentials, nil

}

// SSODeviceCodeFlow contains all the steps to complete a device code flow to retrieve an sso token
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
		StartUrl:     aws.String(rootProfile.AWSConfig.SSOStartURL),
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

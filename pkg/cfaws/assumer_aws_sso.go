package cfaws

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	ssotypes "github.com/aws/aws-sdk-go-v2/service/sso/types"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
	ssooidctypes "github.com/aws/aws-sdk-go-v2/service/ssooidc/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/smithy-go"
	"github.com/common-fate/clio"
	grantedConfig "github.com/common-fate/granted/pkg/config"
	"github.com/common-fate/granted/pkg/securestorage"
	"github.com/hako/durafmt"
	"github.com/pkg/browser"
	"github.com/pkg/errors"
	"gopkg.in/ini.v1"
)

// Implements Assumer
type AwsSsoAssumer struct {
}

func (asa *AwsSsoAssumer) AssumeTerminal(ctx context.Context, c *Profile, configOpts ConfigOpts) (aws.Credentials, error) {
	return c.SSOLogin(ctx, configOpts)
}

func (asa *AwsSsoAssumer) AssumeConsole(ctx context.Context, c *Profile, configOpts ConfigOpts) (aws.Credentials, error) {
	return c.SSOLogin(ctx, configOpts)
}

func (asa *AwsSsoAssumer) Type() string {
	return "AWS_SSO"
}

// Matches the profile type on whether it is an sso profile by checking for ssoaccountid.
func (asa *AwsSsoAssumer) ProfileMatchesType(rawProfile *ini.Section, parsedProfile config.SharedConfig) bool {
	return parsedProfile.SSOAccountID != ""
}

func (c *Profile) SSOLoginWithToken(ctx context.Context, cfg *aws.Config, accessToken *string, secureSSOTokenStorage securestorage.SSOTokensSecureStorage, configOpts ConfigOpts) (aws.Credentials, error) {
	rootProfile := c
	requiresAssuming := false
	if len(c.Parents) > 0 {
		rootProfile = c.Parents[0]

		requiresAssuming = true
	}

	ssoTokenKey := rootProfile.SSOStartURL()
	cfg.Region = rootProfile.SSORegion()
	// create sso client
	ssoClient := sso.NewFromConfig(*cfg)
	var res *ssotypes.RoleCredentials

	if configOpts.ShouldRetryAssuming != nil && *configOpts.ShouldRetryAssuming {
		roleCredentials, err := c.getRoleCredentialsWithRetry(ctx, ssoClient, accessToken, rootProfile)
		if err != nil {
			var unauthorised *ssotypes.UnauthorizedException
			if errors.As(err, &unauthorised) {
				// possible error with the access token we used, in this case we should clear our cached token and request a new one if the user tries again
				secureSSOTokenStorage.ClearSSOToken(ssoTokenKey)
			}
			return aws.Credentials{}, err
		}
		res = roleCredentials
	} else {
		role, err := ssoClient.GetRoleCredentials(ctx, &sso.GetRoleCredentialsInput{AccessToken: accessToken, AccountId: &rootProfile.AWSConfig.SSOAccountID, RoleName: &rootProfile.AWSConfig.SSORoleName})
		if err != nil {
			serr, ok := err.(*smithy.OperationError)
			if ok {
				// If the err is of type ForbiddenRequest then user may be able
				// to request access to the role if they are using Granted Approvals.
				// Display an error message with the request URL, or a prompt
				// to set up the request URL if it's empty.
				if httpErr, ok := serr.Err.(*awshttp.ResponseError); ok {
					if httpErr.HTTPStatusCode() == http.StatusForbidden {
						if c.RawConfig != nil && hasGrantedSSOPrefix(c.RawConfig) {
							gConf, loadErr := grantedConfig.Load()
							if loadErr != nil {
								clio.Debugf(errors.Wrapf(err, "loading Granted config during sso error handling: %s", loadErr.Error()).Error())
								return aws.Credentials{}, serr
							}

							// granted exp request latest will try to auto assume after the request is approved.
							// It is possible that user might still get forbidden access due to some latency is provisioning sso credentials.
							// In such case, let's retry the logic before showing them this error
							// if configOpts.ShouldRetryAssuming != nil && *configOpts.ShouldRetryAssuming {
							// 	fmt.Println("add retry logic here")
							// }
							return aws.Credentials{}, FormatAWSErrorWithGrantedApprovalsURL(serr, c.RawConfig, *gConf, c.AWSConfig.SSORoleName, c.AWSConfig.SSOAccountID)
						}
					}
				}

			}

			var unauthorised *ssotypes.UnauthorizedException
			if errors.As(err, &unauthorised) {
				// possible error with the access token we used, in this case we should clear our cached token and request a new one if the user tries again
				secureSSOTokenStorage.ClearSSOToken(ssoTokenKey)
			}
			return aws.Credentials{}, err
		}

		res = role.RoleCredentials
	}

	rootCreds := TypeRoleCredsToAwsCreds(*res)
	credProvider := &CredProv{rootCreds}

	if requiresAssuming {
		// return creds, nil
		toAssume := append([]*Profile{}, c.Parents[1:]...)
		toAssume = append(toAssume, c)
		for i, p := range toAssume {
			region, err := c.Region(ctx)
			if err != nil {
				return aws.Credentials{}, err
			}
			// in order to support profiles which do not specify a region, we use the default region when assuming the role
			stsClient := sts.New(sts.Options{Credentials: aws.NewCredentialsCache(credProvider), Region: region})
			stsp := stscreds.NewAssumeRoleProvider(stsClient, p.AWSConfig.RoleARN, func(aro *stscreds.AssumeRoleOptions) {
				// all configuration goes in here for this profile
				if p.AWSConfig.RoleSessionName != "" {
					aro.RoleSessionName = p.AWSConfig.RoleSessionName
				} else {
					aro.RoleSessionName = sessionName()
				}
				if p.AWSConfig.MFASerial != "" {
					aro.SerialNumber = &p.AWSConfig.MFASerial
					aro.TokenProvider = MfaTokenProvider
				} else if c.AWSConfig.MFASerial != "" {
					aro.SerialNumber = &c.AWSConfig.MFASerial
					aro.TokenProvider = MfaTokenProvider
				}
				aro.Duration = configOpts.Duration
				if p.AWSConfig.ExternalID != "" {
					aro.ExternalID = &p.AWSConfig.ExternalID
				}
			})
			stsCreds, err := stsp.Retrieve(ctx)
			if err != nil {
				return aws.Credentials{}, err
			}
			// only print for sub assumes because the final credentials are printed at the end of the assume command
			// this is here for visibility into role traversals when assuming a final profile with sso
			if i < len(toAssume)-1 {
				durationDescription := durafmt.Parse(time.Until(stsCreds.Expires) * time.Second).LimitFirstN(1).String()
				clio.Successf("Assumed parent profile: [%s](%s) session credentials will expire %s", p.Name, region, durationDescription)
			}
			credProvider = &CredProv{stsCreds}

		}
	}
	return credProvider.Credentials, nil
}

func (c *Profile) SSOLogin(ctx context.Context, configOpts ConfigOpts) (aws.Credentials, error) {
	rootProfile := c
	if len(c.Parents) > 0 {
		rootProfile = c.Parents[0]
	}
	ssoTokenKey := rootProfile.SSOStartURL() + c.AWSConfig.SSOSessionName
	// if the profile has an sso user configured then suffix the sso token storage key to ensure unique logins
	secureSSOTokenStorage := securestorage.NewSecureSSOTokenStorage()
	cachedToken := secureSSOTokenStorage.GetValidSSOToken(ssoTokenKey)
	// check if profile has a valid plaintext sso access token
	plainTextToken := GetValidSSOTokenFromPlaintextCache(rootProfile.SSOStartURL())

	// store token to storage to avoid multiple logins
	if plainTextToken != nil {
		secureSSOTokenStorage.StoreSSOToken(ssoTokenKey, *plainTextToken)
	}

	var accessToken *string

	skipAutoLogin := configOpts.UsingCredentialProcess && !configOpts.CredentialProcessAutoLogin
	if skipAutoLogin && cachedToken == nil {
		cmd := "granted sso login"
		startURL := c.SSOStartURL()
		if startURL != "" {
			cmd += " --sso-start-url " + startURL
		}

		region := c.SSORegion()
		if region != "" {
			cmd += " --sso-region " + region
		}

		return aws.Credentials{}, fmt.Errorf("error when retrieving credentials from custom process. please login using '%s'", cmd)
	}

	if cachedToken == nil && plainTextToken == nil {
		newCfg := aws.NewConfig()
		newCfg.Region = rootProfile.SSORegion()
		newSSOToken, err := SSODeviceCodeFlowFromStartUrl(ctx, *newCfg, rootProfile.SSOStartURL())
		if err != nil {
			return aws.Credentials{}, err
		}

		secureSSOTokenStorage.StoreSSOToken(ssoTokenKey, *newSSOToken)
		cachedToken = newSSOToken
	}

	if cachedToken != nil {
		accessToken = &cachedToken.AccessToken
	} else {
		accessToken = &plainTextToken.AccessToken
	}

	cfg := aws.NewConfig()
	cfg.Region = c.SSORegion()

	return c.SSOLoginWithToken(ctx, cfg, accessToken, secureSSOTokenStorage, configOpts)
}

func (c *Profile) getRoleCredentialsWithRetry(ctx context.Context, ssoClient *sso.Client, accessToken *string, rootProfile *Profile) (*ssotypes.RoleCredentials, error) {
	maxRetry := 5
	var er error
	for i := 0; i < maxRetry; i++ {
		res, err := ssoClient.GetRoleCredentials(ctx, &sso.GetRoleCredentialsInput{AccessToken: accessToken, AccountId: &rootProfile.AWSConfig.SSOAccountID, RoleName: &rootProfile.AWSConfig.SSORoleName})
		if err == nil {
			return res.RoleCredentials, nil
		} else {
			serr, ok := err.(*smithy.OperationError)
			if ok {
				// If the err is of type ForbiddenRequest then retry
				if httpErr, ok := serr.Err.(*awshttp.ResponseError); ok {
					if httpErr.HTTPStatusCode() == http.StatusForbidden {
						clio.Debugf("failed assuming attempt %d", i)
						// Increase the backoff duration by a second each time we retry
						time.Sleep(time.Second * time.Duration(i+1))
					}
				}
			} else {
				return nil, err
			}

			er = err
		}
	}

	return nil, errors.Wrap(er, "max retries exceeded")
}

// SSODeviceCodeFlowFromStartUrl contains all the steps to complete a device code flow to retrieve an SSO token
func SSODeviceCodeFlowFromStartUrl(ctx context.Context, cfg aws.Config, startUrl string) (*securestorage.SSOToken, error) {
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
		StartUrl:     aws.String(startUrl),
	})
	if err != nil {
		return nil, err
	}

	// trigger OIDC login. open browser to login. close tab once login is done. press enter to continue
	url := aws.ToString(deviceAuth.VerificationUriComplete)
	clio.Info("If the browser does not open automatically, please open this link: " + url)

	// check if sso browser path is set
	config, err := grantedConfig.Load()
	if err != nil {
		return nil, err
	}

	if config.CustomSSOBrowserPath != "" {
		cmd := exec.Command(config.CustomSSOBrowserPath, url)
		err = cmd.Start()
		if err != nil {
			// fail silently
			clio.Debug(err.Error())
		} else {
			// detach from this new process because it continues to run
			err = cmd.Process.Release()
			if err != nil {
				// fail silently
				clio.Debug(err.Error())
			}
		}
	} else {
		err = browser.OpenURL(url)
		if err != nil {
			// fail silently
			clio.Debug(err.Error())
		}
	}

	clio.Info("Awaiting AWS authentication in the browser")
	clio.Info("You will be prompted to authenticate with AWS in the browser, then you will be prompted to 'Allow'")
	clio.Infof("Code: %s", *deviceAuth.UserCode)

	pc := getPollingConfig(deviceAuth)

	token, err := PollToken(ctx, ssooidcClient, *register.ClientSecret, *register.ClientId, *deviceAuth.DeviceCode, pc)
	if err != nil {
		return nil, err
	}

	return &securestorage.SSOToken{AccessToken: *token.AccessToken, Expiry: time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)}, nil
}

var ErrTimeout error = errors.New("polling for device authorization token timed out")

type PollingConfig struct {
	CheckInterval time.Duration
	TimeoutAfter  time.Duration
}

func getPollingConfig(deviceAuth *ssooidc.StartDeviceAuthorizationOutput) PollingConfig {
	return PollingConfig{
		CheckInterval: time.Duration(deviceAuth.Interval) * time.Second,
		TimeoutAfter:  time.Duration(deviceAuth.ExpiresIn) * time.Second,
	}
}

// PollToken will poll for a token and return it once the authentication/authorization flow has been completed in the browser
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

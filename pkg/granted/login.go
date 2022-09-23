package granted

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/common-fate/granted/pkg/cfaws"
	"github.com/urfave/cli/v2"
)

type SSOPlainTextOut struct {
	AccessToken string `json:"accessToken"`
	ExpiresAt   string `json:"expiresAt"`
	StartUrl    string `json:"startUrl"`
	Region      string `json:"region"`
}

var LoginCommand = cli.Command{
	Name:  "login",
	Usage: "Authenticate with AWS SSO for provided profile name",
	Flags: []cli.Flag{&cli.StringFlag{Name: "profile", Required: true}},
	Action: func(c *cli.Context) error {

		profileName := c.String("profile")
		profiles, err := cfaws.LoadProfiles()
		if err != nil {
			return err
		}

		profile, err := profiles.Profile(profileName)
		if err != nil {
			return err
		}

		if err = cfaws.IsValidGrantedProfile(profile.RawConfig); err != nil {
			return err
		}

		gConfig := cfaws.NewGrantedConfig(profile.RawConfig)

		awsConfig, err := gConfig.ConvertToAWSConfig(c.Context, profile)
		if err != nil {
			return err
		}

		profile.AWSConfig = *awsConfig
		cfg := aws.NewConfig()
		cfg.Region = awsConfig.SSORegion

		token, err := cfaws.SSODeviceCodeFlowFromStartUrl(c.Context, *cfg, awsConfig.SSOStartURL, false)
		if err != nil {
			return err
		}

		ssoToken := cfaws.CreatePlainTextSSO(*awsConfig, token)

		if err := ssoToken.DumpToCacheDirectory(); err != nil {
			return err
		}

		fmt.Println("Successfully logged in")

		return nil
	},
}

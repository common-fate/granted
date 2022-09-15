package granted

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/common-fate/granted/pkg/cfaws"
	"github.com/urfave/cli/v2"
)

type grantedSSOCreds struct {
	granted_sso_start_url    string
	granted_sso_region       string
	granted_sso_account_name string
	granted_sso_account_id   string
	granted_sso_role_name    string
	region                   string
}

type SSOPlainTextOut struct {
	AccessToken string `json:"accessToken"`
	ExpiresAt   string `json:"expiresAt"`
	StartUrl    string `json:"startUrl"`
	Region      string `json:"region"`
}

var LoginCommand = cli.Command{
	Name:  "login",
	Usage: "authenticate with AWS SSO for provided profile name",
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

		if err = profile.IsValidGrantedProfile(); err != nil {
			return err
		}

		grantedSSOCreds := grantedSSOCreds{
			granted_sso_start_url:    profile.RawConfig["granted_sso_start_url"],
			granted_sso_region:       profile.RawConfig["granted_sso_region"],
			granted_sso_account_name: profile.RawConfig["granted_sso_account_name"],
			granted_sso_account_id:   profile.RawConfig["granted_sso_account_id"],
			granted_sso_role_name:    profile.RawConfig["granted_sso_role_name"],
			region:                   profile.RawConfig["region"],
		}

		awsConfig, err := convertGrantedCredToAWSConfig(c.Context, profile, grantedSSOCreds)
		if err != nil {
			return err
		}

		profile.AWSConfig = *awsConfig

		cfg := aws.NewConfig()
		cfg.Region = grantedSSOCreds.granted_sso_region

		ssoToken, err := cfaws.SSODeviceCodeFlow(c.Context, *cfg, profile)
		if err != nil {
			return err
		}

		awsCacheOutput := &SSOPlainTextOut{
			AccessToken: ssoToken.AccessToken,
			ExpiresAt:   ssoToken.Expiry.Format(time.RFC3339),
			Region:      grantedSSOCreds.region,
			StartUrl:    grantedSSOCreds.granted_sso_start_url,
		}

		jsonOut, err := json.Marshal(awsCacheOutput)
		if err != nil {
			log.Fatalln("\nUnhandled error when unmarshalling json creds")
		}

		err = dumpTokenFile(jsonOut, grantedSSOCreds.granted_sso_start_url)
		if err != nil {
			return err
		}

		fmt.Println("Successfully logged in.")

		return nil
	},
}

func convertGrantedCredToAWSConfig(ctx context.Context, p *cfaws.Profile, gConfig grantedSSOCreds) (*config.SharedConfig, error) {
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

func getCacheFileName(url string) (string, error) {
	hash := sha1.New()
	_, err := hash.Write([]byte(url))
	if err != nil {
		return "", err
	}
	return strings.ToLower(hex.EncodeToString(hash.Sum(nil))) + ".json", nil
}

func dumpTokenFile(jsonToken []byte, url string) error {
	key, err := getCacheFileName(url)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath.Join(defaultCacheLocation(), key), jsonToken, 0644)
	if err != nil {
		return err
	}

	return nil
}

func defaultCacheLocation() string {
	return filepath.Join(getHomeDirectory(), ".aws", "sso", "cache")
}

// FIXME: Need to check if os is linux or windows. This won't work for windows.
func getHomeDirectory() string {
	return os.Getenv("HOME")
}

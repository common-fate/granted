package assume

import (
	"errors"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/common-fate/granted/pkg/cfaws"
	cfflags "github.com/common-fate/granted/pkg/urfav_overrides"
	"github.com/urfave/cli/v2"
)

// SSOProfileFromFlags will prepare a profile to be assumed from cli flags
func SSOProfileFromFlags(c *cli.Context) (*cfaws.Profile, error) {
	err := ValidateSSOFlags(c)
	if err != nil {
		return nil, err
	}
	s := &cfaws.AwsSsoAssumer{}
	ssoStartURL, ssoRegion, accountID, roleName := ssoFlags(c)
	p := &cfaws.Profile{
		Name:        roleName,
		ProfileType: s.Type(),
		AWSConfig: config.SharedConfig{
			SSOAccountID: accountID,
			SSORoleName:  roleName,
			SSORegion:    ssoRegion,
			SSOStartURL:  ssoStartURL,
		},
		Initialised: true,
	}
	return p, nil
}

func ssoFlags(c *cli.Context) (ssoStartURL, ssoRegion, accountID, roleName string) {
	assumeFlags, err := cfflags.New("assumeFlags", GlobalFlags(), c)
	if err != nil {
		return
	}
	ssoStartURL = assumeFlags.String("sso-start-url")
	ssoRegion = assumeFlags.String("sso-region")
	accountID = assumeFlags.String("account-id")
	roleName = assumeFlags.String("role-name")
	return
}
func ValidateSSOFlags(c *cli.Context) error {
	ssoStartURL, ssoRegion, accountID, roleName := ssoFlags(c)
	if c.Bool("sso") {
		// fmt.Fprintln(os.Stderr, "HELLO")
		// fmt.Fprintln(os.Stderr, ssoStartURL, ssoRegion, accountID, roleName)
		good := ssoStartURL != "" && ssoRegion != "" && accountID != "" && roleName != ""
		if !good {
			return errors.New("flags [sso-start-url, sso-region, account-id, role-name] are required to use the -sso flag")
		}
	} else if ssoStartURL != "" || ssoRegion != "" || accountID != "" || roleName != "" {
		return errors.New("flags [sso-start-url, sso-region, account-id, role-name] can only be used with the -sso flag")
	}
	return nil
}

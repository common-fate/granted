package cfaws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/common-fate/clio"
	"github.com/common-fate/granted/pkg/securestorage"
	"github.com/fatih/color"
	"gopkg.in/ini.v1"
)

const GRANTED_OKTA_INI_OPEN_BROWSER = "granted_okta_open_browser"

type AwsGimmeAwsCredsAssumer struct {
	config           *ini.File
	forceOpenBrowser bool
}

type CredentialCapture struct {
	result *AwsGimmeResult
}

type AwsGimmeResult struct {
	Credentials AwsGimmeCredentials `json:"credentials"`
}

type AwsGimmeCredentials struct {
	AccessKeyID     string `json:"aws_access_key_id"`
	SecretAccessKey string `json:"aws_secret_access_key"`
	SessionToken    string `json:"aws_session_token"`
	Expiration      string `json:"expiration"`
}

func (cc *CredentialCapture) Write(p []byte) (n int, err error) {
	var dest AwsGimmeResult
	err = json.Unmarshal(p, &dest)
	if err != nil {
		return fmt.Fprint(color.Error, string(p))
	}
	cc.result = &dest
	return len(p), nil
}

func (cc *CredentialCapture) Creds() (aws.Credentials, error) {
	if cc.result == nil {
		return aws.Credentials{}, fmt.Errorf("no credential output from gimme-aws-creds")
	}
	c := aws.Credentials{
		AccessKeyID:     cc.result.Credentials.AccessKeyID,
		SecretAccessKey: cc.result.Credentials.SecretAccessKey,
		SessionToken:    cc.result.Credentials.SessionToken,
		Source:          "gimme-aws-creds",
	}
	if cc.result.Credentials.Expiration != "" {
		c.CanExpire = true
		t, err := time.Parse(time.RFC3339, cc.result.Credentials.Expiration)
		if err != nil {
			return aws.Credentials{}, fmt.Errorf("could not parse credentials expiry: %s", cc.result.Credentials.Expiration)
		}
		c.Expires = t
	}
	return c, nil
}

func (gimme *AwsGimmeAwsCredsAssumer) AssumeTerminal(ctx context.Context, c *Profile, configOpts ConfigOpts) (aws.Credentials, error) {
	// try cache
	sessionCredStorage := securestorage.NewSecureSessionCredentialStorage()
	creds, err := sessionCredStorage.GetCredentials(c.AWSConfig.Profile)

	if err != nil {
		clio.Debugw("error loading cached credentials", "error", err)
	} else if creds != nil && !creds.Expired() {
		clio.Debugw("credentials found in cache", "expires", creds.Expires.String(), "canExpire", creds.CanExpire, "timeNow", time.Now().String())
		return *creds, nil
	}

	// if cred process, check we can do a non-interactive refresh
	if configOpts.UsingCredentialProcess {
		err := gimme.LoadGimmeConfig()
		if err != nil {
			return aws.Credentials{}, fmt.Errorf("Failed to load gimme config file: %w", err)
		}
		if !gimme.CanRefreshHeadless(c.Name) {
			return aws.Credentials{}, fmt.Errorf("Cannot refresh Gimme AWS creds in credential_process when force_classic is set.")
		}
		gimme.forceOpenBrowser = true
	}

	clio.Debugw("refreshing credentials", "reason", "none cached")

	// request for the creds if they are invalid
	args := []string{
		fmt.Sprintf("--profile=%s", c.Name),
		"--output-format=json",
	}

	if c.RawConfig.HasKey(GRANTED_OKTA_INI_OPEN_BROWSER) {
		ob, err := c.RawConfig.GetKey(GRANTED_OKTA_INI_OPEN_BROWSER)
		if err != nil {
			clio.Debugf("Error reading ini key %s: %w", GRANTED_OKTA_INI_OPEN_BROWSER, err)
		}

		if ob.MustBool(false) == true || ob.String() == "true" {
			gimme.forceOpenBrowser = true
		}
	}

	if gimme.forceOpenBrowser {
		args = append(args, "--open-browser")
	}

	// add passthrough args
	args = append(args, configOpts.Args...)

	cmd := exec.Command("gimme-aws-creds", args...)

	capture := &CredentialCapture{}
	cmd.Stdout = capture
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr

	cleanEnv := []string{}
	var disallowedVar bool
	for _, env := range os.Environ() {
		disallowedVar = false
		for _, disallowed := range []string{
			"AWS_PROFILE",
			"AWS_ACCESS_KEY_ID",
			"AWS_SECRET_ACCESS_KEY",
			"AWS_SESSION_TOKEN",
			"AWS_REGION",
			"AWS_DEFAULT_REGION",
		} {
			if strings.HasPrefix(env, disallowed) {
				clio.Debugw("removing from exec env", "var", env)
				disallowedVar = true
				break
			}
		}
		if !disallowedVar {
			cleanEnv = append(cleanEnv, env)
		}
	}
	cmd.Env = cleanEnv

	err = cmd.Run()
	if err != nil {
		return aws.Credentials{}, err
	}

	awscreds, err := capture.Creds()
	if err != nil {
		return aws.Credentials{}, err
	}

	// store cached creds
	if err := sessionCredStorage.StoreCredentials(c.AWSConfig.Profile, awscreds); err != nil {
		clio.Warnf("Error caching credentials, MFA token will be requested")
	}

	return awscreds, nil
}

func (gimme *AwsGimmeAwsCredsAssumer) AssumeConsole(ctx context.Context, c *Profile, configOpts ConfigOpts) (aws.Credentials, error) {
	return gimme.AssumeTerminal(ctx, c, configOpts)
}

func (gimme *AwsGimmeAwsCredsAssumer) Type() string {
	return "AWS_GIMME_AWS_CREDS"
}

// parse the gimme config file to check if we have a matching profile
func (gimme *AwsGimmeAwsCredsAssumer) ProfileMatchesType(rawProfile *ini.Section, parsedProfile config.SharedConfig) bool {
	err := gimme.LoadGimmeConfig()
	if errors.Is(err, os.ErrNotExist) {
		return false
	}
	if err != nil {
		clio.Error("Failed to load gimme config file: ", err)
		return false
	}

	for _, section := range gimme.config.SectionStrings() {
		if section == parsedProfile.Profile {
			clio.Debug("matched gimme profile ", section)
			return true
		}
	}

	//clio.Debug("No gimme profile matched")
	return false
}

func (gimme *AwsGimmeAwsCredsAssumer) LoadGimmeConfig() error {
	okta_config := os.Getenv("OKTA_CONFIG")
	if okta_config == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			clio.Error(err)
		}
		okta_config = fmt.Sprintf("%s/.okta_aws_login_config", home)
	}

	_, err := os.Stat(okta_config)
	if err != nil {
		return err
	}

	gimme.config, err = ini.Load(okta_config)
	if err != nil {
		return err
	}

	return nil
}

func (gimme *AwsGimmeAwsCredsAssumer) CanRefreshHeadless(profile string) bool {
	section, err := gimme.config.GetSection(profile)
	if err != nil {
		clio.Warn(err)
		return false
	}

	if section.HasKey("force_classic") {
		key, err := section.GetKey("force_classic")
		if err != nil {
			clio.Warn(err)
			return false
		}
		if key.MustBool(false) == true {
			return false
		}
	}

	if section.HasKey("inherits") {
		key, err := section.GetKey("inherits")
		if err != nil {
			clio.Warn(err)
			return false
		}
		parent := key.MustString("")
		if parent != "" {
			return gimme.CanRefreshHeadless(parent)
		}
	}

	return true
}
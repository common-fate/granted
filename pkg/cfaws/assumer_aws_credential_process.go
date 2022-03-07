package cfaws

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/bigkevmcd/go-configparser"
)

// Implements Assumer using the aws credential_process standard
type CredentialProcessAssumer struct {
}

type CredentialProcessJson struct {
	Version         int
	AccessKeyId     string
	SecretAccessKey string
	SessionToken    string
	Expiration      string
}

// CredentialCapture implements the io.Writer interface and provides a means to capture the credential output from a credential_process call
// as specified by aws documentation https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-sourcing-external.html
type CredentialCapture struct {
	creds *CredentialProcessJson
}

func (wr *CredentialCapture) Write(p []byte) (n int, err error) {
	var dest CredentialProcessJson
	err = json.Unmarshal(p, &dest)
	if err != nil {
		return fmt.Fprint(os.Stderr, string(p))
	}
	wr.creds = &dest
	return len(p), nil
}

func (wr *CredentialCapture) Creds() (aws.Credentials, error) {
	if wr.creds == nil {
		return aws.Credentials{}, fmt.Errorf("no credential output from credential_process")
	}
	c := aws.Credentials{AccessKeyID: wr.creds.AccessKeyId, SecretAccessKey: wr.creds.SecretAccessKey, SessionToken: wr.creds.SessionToken}
	if wr.creds.Expiration != "" {
		c.CanExpire = true
		t, err := time.Parse(time.RFC3339, wr.creds.Expiration)
		if err != nil {
			return aws.Credentials{}, fmt.Errorf("could not parse credentials expiry: %s", wr.creds.Expiration)
		}
		c.Expires = t
	}
	return c, nil

}

type Writer interface {
	Write(p []byte) (n int, err error)
}

func (cpa *CredentialProcessAssumer) AssumeTerminal(ctx context.Context, c *CFSharedConfig, args2 []string) (aws.Credentials, error) {
	var args []string
	var command string
	for k, v := range c.RawConfig {
		if k == "credential_process" {
			s := strings.Split(v, " ")
			command = s[0]
			args = s[1:]
			break
		}
	}

	// https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-sourcing-external.html
	// attempt to run the credential process for this profile
	cmd := exec.Command(command, args...)
	capture := &CredentialCapture{}
	cmd.Stdout = capture
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return aws.Credentials{}, err
	}

	return capture.Creds()
}

func (cpa *CredentialProcessAssumer) AssumeConsole(ctx context.Context, c *CFSharedConfig, args []string) (aws.Credentials, error) {
	return cpa.AssumeTerminal(ctx, c, args)
}

// A unique key which identifies this assumer e.g AWS-SSO or GOOGLE-AWS-AUTH
func (cpa *CredentialProcessAssumer) Type() string {
	return "AWS_CREDENTIAL_PROCESS"
}

// inspect for any credential processes with the saml2aws tool
func (cpa *CredentialProcessAssumer) ProfileMatchesType(rawProfile configparser.Dict, parsedProfile config.SharedConfig) bool {
	for k := range rawProfile {
		if k == "credential_process" {
			return true
		}
	}
	return false
}

package cfaws

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	ssotypes "github.com/aws/aws-sdk-go-v2/service/sso/types"
	"github.com/aws/aws-sdk-go-v2/service/sts/types"
)

func TypeCredsToAwsCreds(c types.Credentials) aws.Credentials {
	return aws.Credentials{AccessKeyID: *c.AccessKeyId, SecretAccessKey: *c.SecretAccessKey, SessionToken: *c.SessionToken, CanExpire: true, Expires: *c.Expiration}
}
func TypeRoleCredsToAwsCreds(c ssotypes.RoleCredentials) aws.Credentials {
	return aws.Credentials{AccessKeyID: *c.AccessKeyId, SecretAccessKey: *c.SecretAccessKey, SessionToken: *c.SessionToken, CanExpire: true, Expires: time.UnixMilli(c.Expiration)}
}

// CredProv implements the aws.CredentialProvider interface
type CredProv struct{ aws.Credentials }

func (c *CredProv) Retrieve(ctx context.Context) (aws.Credentials, error) {
	return c.Credentials, nil
}

// loads the environment variables and hydrates an aws.config if they are present
func GetEnvCredentials(ctx context.Context) aws.Credentials {
	return aws.Credentials{AccessKeyID: os.Getenv("AWS_ACCESS_KEY_ID"), SecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"), SessionToken: os.Getenv("AWS_SESSION_TOKEN")}
}

func GetCredentialsCreds(ctx context.Context, c *CFSharedConfig) (aws.Credentials, error) {
	//check to see if the creds are already exported
	cfg, err := c.AwsConfig(ctx, false)
	if err != nil {
		return aws.Credentials{}, err
	}
	creds, _ := aws.NewCredentialsCache(cfg.Credentials).Retrieve(ctx)

	//check creds are valid - return them if they are
	if creds.HasKeys() && !creds.Expired() {
		return creds, nil
	}
	return aws.Credentials{}, fmt.Errorf("creds invalid or expired")

}

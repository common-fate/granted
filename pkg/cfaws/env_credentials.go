package cfaws

import (
	"context"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
)

// loads the environment variables and hydrates an aws.config if they are present
func GetEnvCredentials(ctx context.Context) aws.Credentials {
	return aws.Credentials{AccessKeyID: os.Getenv("AWS_ACCESS_KEY_ID"), SecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"), SessionToken: os.Getenv("AWS_SESSION_TOKEN")}
}

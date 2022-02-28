package sim

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
)

func Test(ctx context.Context, creds aws.Credentials) {
	client := iam.New(iam.Options{Credentials: aws.CredentialsProviderFunc(func(c context.Context) (aws.Credentials, error) { return creds, nil })})
	client.SimulatePrincipalPolicy(ctx, &iam.SimulatePrincipalPolicyInput{ActionNames: []string{"S3:GetObject"}, CallerArn: aws.String("arn:aws:iam::385788203919:role/CommonFateAuditRole")})
}

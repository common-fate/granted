package cfgcp

import (
	"context"
	"time"

	credentials "cloud.google.com/go/iam/credentials/apiv1"
	"cloud.google.com/go/iam/credentials/apiv1/credentialspb"
	"google.golang.org/protobuf/types/known/durationpb"
)

// Implements Assumer
type GCPServiceAccountAssumer struct {
}

func (asa *GCPServiceAccountAssumer) AssumeTerminal(ctx context.Context, ServiceAccount *ServiceAccount) (GCPCredentials, error) {

	c, err := credentials.NewIamCredentialsClient(ctx)
	if err != nil {
		return GCPCredentials{}, err
	}
	defer c.Close()

	req := &credentialspb.GenerateAccessTokenRequest{
		Name:     ServiceAccount.Name,
		Lifetime: durationpb.New(time.Hour),
		Scope:    []string{"https://www.googleapis.com/auth/cloud-platform"},
	}
	resp, err := c.GenerateAccessToken(ctx, req)
	if err != nil {
		return GCPCredentials{}, err
	}
	return GCPCredentials{
		AccessToken: resp.AccessToken,
		ExpireTime:  resp.ExpireTime.AsTime().String(),
	}, nil
}

func (asa *GCPServiceAccountAssumer) AssumeConsole(ctx context.Context, ServiceAccount *ServiceAccount) (GCPCredentials, error) {
	return GCPCredentials{}, nil
}

func (asa *GCPServiceAccountAssumer) Type() string {
	return "GCP_SERVICE_ACCOUNT_IMPERSONATION"
}

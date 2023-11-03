package cfgcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	credentials "cloud.google.com/go/iam/credentials/apiv1"
	"cloud.google.com/go/iam/credentials/apiv1/credentialspb"
	"google.golang.org/protobuf/types/known/durationpb"
)

// Implements Assumer
type GCPServiceAccountAssumer struct {
}

type ImpersonatedServiceAccountConfig struct {
	Delegates                      []string `json:"delegates"`
	ServiceAccountImpersonationURL string   `json:"service_account_impersonation_url"`
	AccessToken                    string   `json:"access_token"`
	Type                           string   `json:"type"`
}

func (asa *GCPServiceAccountAssumer) AssumeTerminal(ctx context.Context, ServiceAccount *ServiceAccount) ([]byte, error) {

	c, err := credentials.NewIamCredentialsClient(ctx)
	if err != nil {
		return []byte{}, err
	}
	defer c.Close()

	req := &credentialspb.GenerateAccessTokenRequest{
		Name:     ServiceAccount.Name,
		Lifetime: durationpb.New(time.Hour),
		Scope:    []string{"https://www.googleapis.com/auth/cloud-platform"},
	}
	resp, err := c.GenerateAccessToken(ctx, req)
	if err != nil {
		return []byte{}, err
	}

	b := ImpersonatedServiceAccountConfig{
		AccessToken: resp.AccessToken,
		Type:        "impersonated_service_account",
		ServiceAccountImpersonationURL: fmt.Sprintf(
			"https://iamcredentials.googleapis.com/v1/projects/-/serviceAccounts/%s.iam.gserviceaccount.com:generateAccessToken",
			ServiceAccount.Name),
	}
	// Marshal the struct to JSON
	jsonData, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return nil, err
	}

	return jsonData, nil
}

func (asa *GCPServiceAccountAssumer) AssumeConsole(ctx context.Context, ServiceAccount *ServiceAccount) ([]byte, error) {
	return []byte{}, nil
}

func (asa *GCPServiceAccountAssumer) Type() string {
	return "GCP_SERVICE_ACCOUNT_IMPERSONATION"
}

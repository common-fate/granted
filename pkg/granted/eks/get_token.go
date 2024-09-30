package eks

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sts"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

const (
	clusterIDHeader        = "x-k8s-aws-id"
	presignedURLExpiration = 10 * time.Minute
	v1Prefix               = "k8s-aws-v1."
)

type Token struct {
	Token      string    `json:"token"`
	Expiration time.Time `json"expiration"`
}

type ExecCredential struct {
	ApiVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Status     Token  `json:"status"`
}

func getExecAuth(token Token) (string, error) {
	execAuth := ExecCredential{
		ApiVersion: "client.authentication.k8s.io/v1beta1",
		Kind:       "ExecCredential",
		Status:     token,
	}
	encoded, err := json.MarshalIndent(execAuth, "", "  ")
	return string(encoded), err
}

func getToken(ctx context.Context, client *sts.Client, clusterName string) (Token, error) {
	pclient := sts.NewPresignClient(client)
	// generate a sts:GetCallerIdentity request and add our custom cluster ID header
	presignedURLRequest, err := pclient.PresignGetCallerIdentity(ctx, &sts.GetCallerIdentityInput{}, func(presignOptions *sts.PresignOptions) {
		presignOptions.ClientOptions = append(presignOptions.ClientOptions, appendPresignHeaderValuesFunc(clusterName))
	})
	if err != nil {
		return Token{}, fmt.Errorf("failed to presign caller identity: %w", err)
	}
	tokenExpiration := time.Now().Local().Add(presignedURLExpiration)
	// Add the token with k8s-aws-v1. prefix.
	return Token{v1Prefix + base64.RawURLEncoding.EncodeToString([]byte(presignedURLRequest.URL)), tokenExpiration}, nil
}

func appendPresignHeaderValuesFunc(clusterID string) func(stsOptions *sts.Options) {
	return func(stsOptions *sts.Options) {
		stsOptions.APIOptions = append(stsOptions.APIOptions,
			// Add clusterId Header.
			smithyhttp.SetHeaderValue(clusterIDHeader, clusterID),
			// Add X-Amz-Expires query param.
			smithyhttp.SetHeaderValue("X-Amz-Expires", "60"))
	}
}

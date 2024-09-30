package eks

import (
	"github.com/common-fate/sdk/eid"
)

// AWSEKS is the grant output for AWS Resource-Based Access Grants.
//
// AWSEKS is excluded from the Cedar schema.
type AWSEKSGrantOutput struct {
	ID               string `json:"id" authz:"id"`
	SSOStartURL      string `json:"sso_start_url" authz:"sso_start_url"`
	SSORoleName      string `json:"sso_role_name" authz:"sso_role_name"`
	SSORegion        string `json:"sso_region" authz:"sso_region"`
	AWSAccountID     string `json:"aws_account_id" authz:"aws_account_id"`
	PermissionSetARN string `json:"permission_set_arn" authz:"permission_set_arn"`
	ClusterRegion    string `json:"cluster_region" authz:"cluster_region"`
}

func (e AWSEKSGrantOutput) EID() eid.EID { return eid.New(EKSGrantOutputType, e.ID) }

const EKSGrantOutputType = "CF::GrantOutput::AWSEKS"

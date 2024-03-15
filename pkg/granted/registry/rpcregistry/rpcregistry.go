package rpcregistry

import (
	"context"

	"github.com/common-fate/clio"
	"gopkg.in/ini.v1"
)

type Registry struct {
}

const EXAMPLE_PROFILES = `
[profile tax-api-prod/S3ListBuckets]
granted_sso_start_url      = https://d-976708da7d.awsapps.com/start
granted_sso_region         = ap-southeast-2
granted_sso_account_id     = 179203389603
granted_sso_role_name      = S3ListBuckets
common_fate_generated_from = common-fate-v1
credential_process         = granted credential-process --profile tax-api-prod/S3ListBuckets

[profile Sandbox-2/AWSAdministratorAccess]
granted_sso_start_url      = https://d-976708da7d.awsapps.com/start
granted_sso_region         = ap-southeast-2
granted_sso_account_id     = 089414121703
granted_sso_role_name      = AWSReadOnlyAccess
common_fate_generated_from = common-fate-v1
credential_process         = granted credential-process --profile Sandbox-2/AWSAdministratorAccess
`

func (r Registry) AWSProfiles(ctx context.Context) (*ini.File, error) {
	data, err := ini.Load([]byte(EXAMPLE_PROFILES))
	if err != nil {
		return nil, err
	}

	clio.Infof("synced available AWS profiles from Common Fate")

	return data, nil
}

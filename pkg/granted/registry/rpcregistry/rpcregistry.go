package rpcregistry

import (
	"context"
	"time"

	"github.com/common-fate/clio"
	"gopkg.in/ini.v1"
)

type Registry struct {
}

const EXAMPLE_PROFILES = `
[profile tax-api-prod/ViewOnlyAccess]
granted_sso_start_url      = https://d-976708da7d.awsapps.com/start
granted_sso_region         = ap-southeast-2
granted_sso_account_id     = 179203389603
granted_sso_role_name      = ViewOnlyAccess
common_fate_generated_from = common-fate-v1
credential_process         = granted credential-process --profile tax-api-prod/ViewOnlyAccess

[profile Sandbox-2/AWSAdministratorAccess]
granted_sso_start_url      = https://d-976708da7d.awsapps.com/start
granted_sso_region         = ap-southeast-2
granted_sso_account_id     = 089414121703
granted_sso_role_name      = AWSAdministratorAccess
common_fate_generated_from = common-fate-v1
credential_process         = granted credential-process --profile Sandbox-2/AWSAdministratorAccess
`

func (r Registry) AWSProfiles(ctx context.Context) (*ini.File, error) {
	data, err := ini.Load([]byte(EXAMPLE_PROFILES))
	if err != nil {
		return nil, err
	}

	time.Sleep(time.Millisecond * 300)

	clio.Infof("synced available AWS profiles from Common Fate")

	return data, nil
}

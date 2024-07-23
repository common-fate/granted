package rds

import (
	"github.com/common-fate/sdk/eid"
)

const GrantOutputType = "CF::GrantOutput::AWSRDS"

// AWSRDS is the grant output for AWS RDS Grants.
//
// AWSRDS is excluded from the Cedar schema.
type AWSRDS struct {
	Grant            eid.EID      `json:"grant" authz:"grant"`
	Name             string       `json:"name" authz:"name"`
	SSOStartURL      string       `json:"sso_start_url" authz:"sso_start_url"`
	SSORoleName      string       `json:"sso_role_name" authz:"sso_role_name"`
	SSORegion        string       `json:"sso_region" authz:"sso_region"`
	Database         Database     `json:"database" authz:"database"`
	User             DatabaseUser `json:"user" authz:"user"`
	SSMSessionTarget string       `json:"ssm_session_target" authz:"ssm_session_target"`
	PermissionSetARN string       `json:"permission_set_arn" authz:"permission_set_arn"`
}

func (e AWSRDS) Parents() []eid.EID { return []eid.EID{e.Grant} }

func (e AWSRDS) EID() eid.EID { return eid.New(AWSRDSType, e.Grant.ID) }

const AWSRDSType = "CF::GrantOutput::AWSRDS"

type Database struct {
	ID         string  `json:"id" authz:"id"`
	Name       string  `json:"name" authz:"name"`
	Engine     string  `json:"engine" authz:"engine"`
	Region     string  `json:"region" authz:"region"`
	Account    eid.EID `json:"account" authz:"account,relation=AWS::Account"`
	InstanceID string  `json:"instance_id" authz:"instance_id"`
	// the name of the database on the instance, used when connecting
	Database string `json:"database" authz:"database"`
}

func (e Database) EID() eid.EID       { return eid.New(DatabaseType, e.ID) }
func (e Database) Parents() []eid.EID { return []eid.EID{e.Account} }

const DatabaseType = "AWS::RDS::Database"

type DatabaseUser struct {
	ID       string  `json:"id" authz:"id"`
	Name     string  `json:"name" authz:"name"`
	Username string  `json:"username" authz:"username"`
	Database eid.EID `json:"database" authz:"database,relation=AWS::RDS::Database"`
}

func (e DatabaseUser) EID() eid.EID { return eid.New(DatabaseUserType, e.ID) }

func (e DatabaseUser) Parents() []eid.EID { return []eid.EID{e.Database} }

const DatabaseUserType = "AWS::RDS::DatabaseUser"

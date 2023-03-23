package models

type AccessRule struct {
	ID                 string `db:"id"`
	Name               string `db:"name"`
	DeploymentID       string `db:"deployment_id"`
	TargetProviderID   string `db:"target_provider_id"`
	TargetProviderType string `db:"target_provider_type"`
	CreatedAt          int64  `db:"created_at"`
	UpdatedAt          int64  `db:"updated_at"`
	DurationSeconds    int    `db:"duration_seconds"`
	RequiresApproval   int    `db:"requires_approval"`
}

type AccessTarget struct {
	RuleID      string `db:"rule_id"`
	Type        string `db:"type"`
	Label       string `db:"label"`
	Description string `db:"description"`
	Value       string `db:"value"`
}

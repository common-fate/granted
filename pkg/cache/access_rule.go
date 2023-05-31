package cache

type AccessRule struct {
	ID                 string         `json:"id"`
	Name               string         `json:"name"`
	DeploymentID       string         `json:"deployment_id"`
	TargetProviderID   string         `json:"target_provider_id"`
	TargetProviderType string         `json:"target_provider_type"`
	CreatedAt          int64          `json:"created_at"`
	UpdatedAt          int64          `json:"updated_at"`
	DurationSeconds    int            `json:"duration_seconds"`
	RequiresApproval   bool           `json:"requires_approval"`
	Targets            []AccessTarget `json:"targets"`
}

type AccessTarget struct {
	RuleID      string `json:"rule_id"`
	Type        string `json:"type"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Value       string `json:"value"`
}

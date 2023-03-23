CREATE TABLE IF NOT EXISTS cf_access_rules (
    id TEXT NOT NULL PRIMARY KEY,
    deployment_id TEXT NOT NULL,
    name TEXT,
    target_provider_id TEXT,
    target_provider_type TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    duration_seconds INTEGER NOT NULL,
    requires_approval INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS cf_access_targets (
    rule_id TEXT NOT NULL,
    type TEXT NOT NULL,
    label TEXT NOT NULL,
    description TEXT NOT NULL,
    value TEXT NOT NULL
);

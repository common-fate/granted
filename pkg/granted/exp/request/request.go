package request

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/briandowns/spinner"
	"github.com/common-fate/cli/pkg/client"
	cfconfig "github.com/common-fate/cli/pkg/config"
	"github.com/common-fate/clio"
	"github.com/common-fate/common-fate/pkg/types"
	"github.com/common-fate/granted/pkg/cache/models"
	"github.com/common-fate/granted/pkg/config"
	"github.com/common-fate/granted/pkg/securestorage"
	"github.com/hako/durafmt"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
	"golang.org/x/sync/errgroup"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

var Command = cli.Command{
	Name:  "request",
	Usage: "Request access to a role",
	Subcommands: []*cli.Command{
		&awsCommand,
	},
}

var awsCommand = cli.Command{
	Name:  "aws",
	Usage: "Request access to an AWS role",
	Action: func(c *cli.Context) error {
		ctx := c.Context
		db, err := sqlx.Open("sqlite3", "file:granted.db")
		if err != nil {
			return err
		}

		cfcfg, err := cfconfig.Load()
		if err != nil {
			return err
		}

		k, err := securestorage.NewCF().Storage.Keyring()
		if err != nil {
			return errors.Wrap(err, "loading keyring")
		}

		cf, err := client.FromConfig(ctx, cfcfg, client.WithKeyring(k))
		if err != nil {
			return err
		}

		depID := cfcfg.CurrentOrEmpty().DashboardURL

		existingRules, err := getCachedAccessRules(depID)
		if err != nil {
			return err
		}

		clio.Debugw("got cached access rules", "rules", existingRules)

		rules, err := cf.UserListAccessRulesWithResponse(ctx)
		if err != nil {
			return err
		}

		for _, r := range rules.JSON200.AccessRules {
			var g errgroup.Group

			g.Go(func() error {
				return updateCachedAccessRule(ctx, updateCacheOpts{
					Rule:         r,
					Existing:     existingRules,
					DB:           db,
					DeploymentID: depID,
					CF:           cf,
				})
			})

			err = g.Wait()
			if err != nil {
				return err
			}
		}

		// refresh the cache
		existingRules, err = getCachedAccessRules(depID)
		if err != nil {
			return err
		}

		// note: we use a map here to de-duplicate accounts.
		// this means that the RuleID in the accounts map is not necessarily
		// the *only* Access Rule which grants access to that account.
		accounts := map[string]models.AccessTarget{}

		// a map of access rule IDs that match each account ID
		// Prod (123456789012) -> {"rul_123": true}
		accessRulesForAccount := map[string]map[string]bool{}
		// rows, err = db.Queryx("SELECT * FROM cf_access_targets WHERE type = 'accountId'")
		// if err != nil {
		// 	return err
		// }

		// for rows.Next() {
		// 	var t models.AccessTarget
		// 	err := rows.StructScan(&t)
		// 	if err != nil {
		// 		return err
		// 	}
		// 	accounts[t.Value] = t

		// 	if _, ok := accessRulesForAccount[t.Value]; !ok {
		// 		accessRulesForAccount[t.Value] = map[string]bool{}
		// 	}

		// 	accessRulesForAccount[t.Value][t.RuleID] = true
		// }

		// note: we use a map here to de-duplicate accounts.
		// this means that the RuleID in the accounts map is not necessarily
		// the *only* Access Rule which grants access to that account.
		permissionSets := map[string]models.AccessTarget{}

		for _, rule := range existingRules {
			for _, t := range rule.Targets {
				if t.Type == "accountId" {
					if _, ok := accessRulesForAccount[t.Value]; !ok {
						accessRulesForAccount[t.Value] = map[string]bool{}
					}

					accessRulesForAccount[t.Value][rule.ID] = true
				}

				if t.Type == "permissionSetArn" {
					if _, ok := accessRulesForAccount[t.Value]; !ok {
						accessRulesForAccount[t.Value] = map[string]bool{}
					}

					permissionSets[t.Value] = t
				}
			}
		}

		// a mapping of the selected survey prompt option, back to the actual value
		// e.g. "my-account-name (123456789012)" -> 123456789012
		selectedAccountMap := map[string]string{}
		var accountOptions []string
		for _, a := range accounts {
			option := fmt.Sprintf("%s (%s)", a.Label, a.Value)
			accountOptions = append(accountOptions, option)
			selectedAccountMap[option] = a.Value
		}

		var selectedAccountOption string

		prompt := &survey.Select{
			Message: "Account",
			Options: accountOptions,
		}
		err = survey.AskOne(prompt, &selectedAccountOption)
		if err != nil {
			return err
		}

		selectedAccountID := selectedAccountMap[selectedAccountOption]
		selectedAccountInfo := accounts[selectedAccountID]
		ruleIDs := accessRulesForAccount[selectedAccountID]

		// rule IDs to include in the SQL query
		var queryRuleIDs []string
		for ruleID := range ruleIDs {
			queryRuleIDs = append(queryRuleIDs, ruleID)
		}

		// find Access Rules that match the selected account

		// query, args, err := sqlx.In("SELECT * FROM cf_access_targets WHERE type = 'permissionSetArn' AND rule_id IN (?)", queryRuleIDs)
		// if err != nil {
		// 	return err
		// }

		// query = db.Rebind(query)

		// rows, err = db.Queryx(query, args...)
		// if err != nil {
		// 	return err
		// }

		// for rows.Next() {
		// 	var t models.AccessTarget
		// 	err := rows.StructScan(&t)
		// 	if err != nil {
		// 		return err
		// 	}
		// 	permissionSets[t.Value] = t
		// }

		// map of permission set option label to Access Rule ID
		// AdminAccess -> {"rul_123": true}
		permissionSetRuleIDs := map[string]map[string]bool{}

		// map of permission set option label to permission set value
		permissionSetValues := map[string]string{}

		var permissionSetOptions []string
		for _, a := range permissionSets {
			permissionSetOptions = append(permissionSetOptions, a.Label) // label only for permission sets (the ARN is difficult to interpret and the labels are unique)

			if _, ok := permissionSetRuleIDs[a.Label]; !ok {
				permissionSetRuleIDs[a.Label] = map[string]bool{}
			}

			permissionSetRuleIDs[a.Label][a.RuleID] = true
			permissionSetValues[a.Label] = a.Value
		}

		var selectedRole string

		prompt = &survey.Select{
			Message: "Role",
			Options: permissionSetOptions,
		}
		err = survey.AskOne(prompt, &selectedRole)
		if err != nil {
			return err
		}

		permissionSetArn := permissionSetValues[selectedRole]

		selectedPermissionSetRuleIDs := permissionSetRuleIDs[selectedRole]

		// find Access Rules that match the permission set and the account
		// we need to find the intersection between permissionSetRuleIDs and accessRulesForAccount
		// matchingAccessRule tracks the current Access Rule which we'll use to request access against.
		var matchingAccessRule *models.AccessRule

		for ruleID := range ruleIDs {
			if _, ok := selectedPermissionSetRuleIDs[ruleID]; ok {

				// the Access Rule matches both the account and the permission set and could be selected
				rule := existingRules[ruleID]

				clio.Debugw("considering access rule", "rule.proposed", rule, "rule.matched", matchingAccessRule)

				// if we haven't found a match yet, set the matching access rule as this one.
				if matchingAccessRule == nil {
					matchingAccessRule = &rule
					continue
				}

				// if we've found a match, use this rule if it's lesser "resistance" than the existing
				// matched one.

				// the proposed rule will take priority if it doesn't require approval
				if matchingAccessRule.RequiresApproval == 1 && rule.RequiresApproval == 0 {
					matchingAccessRule = &rule
					continue
				}

				// the proposed rule will take priority if it has a longer duration
				if matchingAccessRule.RequiresApproval == rule.RequiresApproval &&
					matchingAccessRule.DurationSeconds < rule.DurationSeconds {
					matchingAccessRule = &rule
					continue
				}
			}
		}

		clio.Debugw("matched access rule", "rule.matched", matchingAccessRule)

		var reason string

		reasonPrompt := &survey.Input{
			Message: "Reason for access:",
			Help:    "Will be stored in audit trails and associated with you",
			Suggest: func(toComplete string) []string {
				return []string{"dev work", "resolving issue for CS-123"}
			},
		}
		err = survey.AskOne(reasonPrompt, &reason)
		if err != nil {
			return err
		}

		clio.NewLine()
		clio.Infof("Run this one-liner command to request access in future:\ngranted request aws --account %s --role %s --reason \"%s\"", selectedAccountID, selectedRole, reason)
		clio.NewLine()

		si := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
		si.Suffix = " requesting access..."
		si.Writer = os.Stderr
		si.Start()

		// the current version of the API requires `With` fields to be provided
		// *only* if the Access Rule has multiple options for that field.
		var with []types.CreateRequestWith

		// check if the 'accountId' field needs to be included
		rows, err = db.Queryx("SELECT COUNT(*) FROM cf_access_targets WHERE type = 'accountId' AND rule_id = $1", matchingAccessRule.ID)
		if err != nil {
			return err
		}
		defer rows.Close()

		var count int
		for rows.Next() {
			if err := rows.Scan(&count); err != nil {
				return err
			}
		}

		if count > 1 {
			with = append(with, types.CreateRequestWith{
				AdditionalProperties: map[string][]string{
					"accountId": {selectedAccountID},
				},
			})
		}

		// check if the 'permissionSetArn' field needs to be included
		rows, err = db.Queryx("SELECT COUNT(*) FROM cf_access_targets WHERE type = 'permissionSetArn' AND rule_id = $1", matchingAccessRule.ID)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			if err := rows.Scan(&count); err != nil {
				return err
			}
		}

		if count > 1 {
			with = append(with, types.CreateRequestWith{
				AdditionalProperties: map[string][]string{
					"permissionSetArn": {permissionSetArn},
				},
			})
		}

		// withPtr is set to null if the `With` field doesn't contain anything.
		// it is used to avoid API bad request errors.
		var withPtr *[]types.CreateRequestWith
		if len(with) > 0 {
			withPtr = &with
		}

		res, err := cf.UserCreateRequestWithResponse(ctx, types.UserCreateRequestJSONRequestBody{
			AccessRuleId: matchingAccessRule.ID,
			Reason:       &reason,
			Timing: types.RequestTiming{
				// use the maximum allowed time on the rule by default
				// to minimise the number of prompts to users.
				DurationSeconds: matchingAccessRule.DurationSeconds,
			},
			With: withPtr,
		})
		if err != nil {
			return err
		}

		si.Stop()

		// should only have a single request here
		for _, r := range res.JSON200.Requests {
			clio.Infof("Access Request %s - %s", r.ID, r.Status)
			reqURL, err := url.Parse(cfcfg.CurrentOrEmpty().DashboardURL)
			if err != nil {
				return err
			}
			reqURL.Path = path.Join("/requests", r.ID)
			clio.Infof("URL: %s", reqURL)

			fullName := fmt.Sprintf("%s/%s", selectedAccountInfo.Label, selectedRole)
			clio.Infof("To use the profile with the AWS CLI, sync your ~/.aws/config by running 'granted sso populate'. Then, run:\nexport AWS_PROFILE=%s", fullName)
			clio.NewLine()

			if r.Status == types.RequestStatusAPPROVED {
				durationDescription := durafmt.Parse(time.Duration(matchingAccessRule.DurationSeconds) * time.Second).LimitFirstN(1).String()
				clio.Successf("[%s] Access is activated (expires in %s)", fullName, durationDescription)
			}
		}

		return nil
	},
}

func getCachedAccessRules(depID string) (map[string]models.AccessRule, error) {
	configFolder, err := config.GrantedConfigFolder()
	if err != nil {
		return nil, err
	}
	depURL, err := url.Parse(depID)
	if err != nil {
		return nil, err
	}

	// ~/.granted/common-fate-cache/commonfate.example.com/access-rules
	cacheFolder := path.Join(configFolder, "common-fate-cache", depURL.Hostname(), "access-rules")

	if _, err := os.Stat(cacheFolder); err == os.ErrNotExist {
		clio.Debugw("cache folder does not exist, returning", "folder", cacheFolder, "error", err)
		return nil, nil
	}

	files, err := os.ReadDir(cacheFolder)
	if err != nil {
		return nil, err
	}

	// map of rule ID to the rule itself
	rules := map[string]models.AccessRule{}

	for _, f := range files {
		// the name of the file is the rule ID (e.g. `rul_123`)
		ruleBytes, err := os.ReadFile(path.Join(cacheFolder, f.Name()))
		if err != nil {
			return nil, err
		}
		var rule models.AccessRule
		err = json.Unmarshal(ruleBytes, &rule)
		if err != nil {
			return nil, err
		}

		rules[f.Name()] = rule
	}

	return rules, nil
}

func getCachedAccessTargets(depID string) (map[string]models.AccessRule, error) {
	configFolder, err := config.GrantedConfigFolder()
	if err != nil {
		return nil, err
	}
	depURL, err := url.Parse(depID)
	if err != nil {
		return nil, err
	}

	// ~/.granted/common-fate-cache/commonfate.example.com/access-targets/aws/account
	cacheFolder := path.Join(configFolder, "common-fate-cache", depURL.Hostname(), "access-targets", "aws", "account")

	if _, err := os.Stat(cacheFolder); err == os.ErrNotExist {
		clio.Debugw("cache folder does not exist, returning", "folder", cacheFolder, "error", err)
		return nil, nil
	}

	files, err := os.ReadDir(cacheFolder)
	if err != nil {
		return nil, err
	}

	// map of rule ID to the rule itself
	rules := map[string]models.AccessRule{}

	for _, f := range files {
		// the name of the file is the rule ID (e.g. `rul_123`)
		ruleBytes, err := os.ReadFile(path.Join(cacheFolder, f.Name()))
		if err != nil {
			return nil, err
		}
		var rule models.AccessRule
		err = json.Unmarshal(ruleBytes, &rule)
		if err != nil {
			return nil, err
		}

		rules[f.Name()] = rule
	}

	return rules, nil
}

type updateCacheOpts struct {
	Rule         types.AccessRule
	Existing     map[string]models.AccessRule
	DB           *sqlx.DB
	DeploymentID string
	CF           *client.Client
}

func updateCachedAccessRule(ctx context.Context, opts updateCacheOpts) error {
	r := opts.Rule
	if opts.Rule.Target.Provider.Type != "aws-sso" {
		clio.Debugw("skipping syncing rule: only aws-sso provider type supported", "rule.provider.type", opts.Rule.Target.Provider.Type)
		return nil
	}

	existing, ok := opts.Existing[r.ID]

	if ok {
		// the rule exists in the cache - check whether it's been updated
		// since we last saw it.
		cacheUpdatedAt := time.Unix(existing.UpdatedAt, 0)
		if !opts.Rule.UpdatedAt.After(opts.Rule.UpdatedAt) {
			clio.Debugw("rule is up to date: skipping sync", "rule.id", r.ID, "cache.updated_at", cacheUpdatedAt.Unix(), "rule.updated_at", opts.Rule.UpdatedAt.Unix())
			return nil
		}
		clio.Debugw("rule is out of date", "rule.id", r.ID, "cache.updated_at", cacheUpdatedAt.Unix(), "rule.updated_at", opts.Rule.UpdatedAt.Unix())
	} else {
		// doesn't exist in the cache
		row := models.AccessRule{
			ID:                 r.ID,
			Name:               r.Name,
			DeploymentID:       opts.DeploymentID,
			TargetProviderID:   r.Target.Provider.Id,
			TargetProviderType: r.Target.Provider.Type,
			CreatedAt:          r.CreatedAt.Unix(),
			UpdatedAt:          r.UpdatedAt.Unix(),
			DurationSeconds:    r.TimeConstraints.MaxDurationSeconds,
		}

		_, err := opts.DB.NamedExecContext(ctx, `INSERT INTO cf_access_rules (id, deployment_id, name, target_provider_id, target_provider_type, created_at, updated_at, duration_seconds, requires_approval) 
			VALUES (:id, :deployment_id, :name, :target_provider_id, :target_provider_type, :created_at, :updated_at, :duration_seconds, :requires_approval)`, &row)
		if err != nil {
			return err
		}
	}

	// our API doesn't easily expose whether manual approval is required
	// on an Access Rule, so we need to fetch approvers separately.

	approvers, err := opts.CF.UserGetAccessRuleApproversWithResponse(ctx, r.ID)
	if err != nil {
		return err
	}

	// SQLite uses an int to store booleans. 0 = false, 1 = true
	var requiresApproval int

	if len(approvers.JSON200.Users) > 0 {
		requiresApproval = 1
	}

	_, err = opts.DB.ExecContext(ctx, "UPDATE cf_access_rules SET requires_approval = $1 WHERE id = $2", requiresApproval, r.ID)
	if err != nil {
		return err
	}

	clio.Debugw("updated requires approval", "rule.id", r.ID, "requires_approval", requiresApproval)

	// if we get here, we need to sync the resources that the rule grants access to
	details, err := opts.CF.UserGetAccessRuleWithResponse(ctx, r.ID)
	if err != nil {
		return err
	}

	var targets []models.AccessTarget

	for k, v := range details.JSON200.Target.Arguments.AdditionalProperties {
		for _, o := range v.Options {
			t := models.AccessTarget{
				RuleID: r.ID,
				Type:   k,
				Label:  o.Label,
				Value:  o.Value,
			}

			if o.Description != nil {
				t.Description = *o.Description
			}
			targets = append(targets, t)
		}
	}

	tx, err := opts.DB.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, "DELETE FROM cf_access_targets WHERE rule_id = $1", r.ID)
	if err != nil {
		return err
	}

	_, err = tx.NamedExec(`INSERT INTO cf_access_targets (rule_id, type, label, description, value)
        VALUES (:rule_id, :type, :label, :description, :value)`, targets)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	clio.Debugw("updated access targets", "rule.id", r.ID, "targets.count", len(targets))

	return nil
}

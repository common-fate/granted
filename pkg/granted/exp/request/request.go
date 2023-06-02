package request

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/briandowns/spinner"
	"github.com/common-fate/cli/pkg/client"
	cfconfig "github.com/common-fate/cli/pkg/config"
	"github.com/common-fate/clio"
	"github.com/common-fate/clio/clierr"
	"github.com/common-fate/common-fate/pkg/types"
	"github.com/common-fate/granted/pkg/accessrequest"
	"github.com/common-fate/granted/pkg/cache"
	"github.com/common-fate/granted/pkg/cfaws"
	"github.com/common-fate/granted/pkg/config"
	"github.com/common-fate/granted/pkg/frecency"
	"github.com/common-fate/granted/pkg/securestorage"
	"github.com/hako/durafmt"
	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
	"golang.org/x/sync/errgroup"
)

var Command = cli.Command{
	Name:  "request",
	Usage: "Request access to a role",
	Subcommands: []*cli.Command{
		&awsCommand,
		&latestCommand,
	},
}

var awsCommand = cli.Command{
	Name:  "aws",
	Usage: "Request access to an AWS role",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "account", Usage: "The AWS account ID"},
		&cli.StringFlag{Name: "role", Usage: "The AWS role"},
		&cli.StringFlag{Name: "reason", Usage: "A reason for access"},
	},
	Action: func(c *cli.Context) error {
		return requestAccess(c.Context, requestAccessOpts{
			account: c.String("account"),
			role:    c.String("role"),
			reason:  c.String("reason"),
		})
	},
}

var latestCommand = cli.Command{
	Name:  "latest",
	Usage: "Request access to the latest AWS role you attempted to use",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "reason", Usage: "A reason for access"},
	},
	Action: func(c *cli.Context) error {
		role, err := accessrequest.LatestRole()
		if err != nil {
			return err
		}

		clio.Infof("requesting access to account %s with role %s", role.Account, role.Role)

		return requestAccess(c.Context, requestAccessOpts{
			account: role.Account,
			role:    role.Role,
			reason:  c.String("reason"),
		})
	},
}

type requestAccessOpts struct {
	account string
	role    string
	reason  string
}

func requestAccess(ctx context.Context, opts requestAccessOpts) error {

	cfcfg, err := cfconfig.Load()
	if err != nil {
		return err
	}

	k, err := securestorage.NewCF().Storage.Keyring()
	if err != nil {
		return errors.Wrap(err, "loading keyring")
	}

	cf, err := client.FromConfig(ctx, cfcfg, client.WithKeyring(k), client.WithLoginHint("granted login"))
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

	clio.Debugw("refreshed cache", "rules", existingRules)

	// note: we use a map here to de-duplicate accounts.
	// this means that the RuleID in the accounts map is not necessarily
	// the *only* Access Rule which grants access to that account.
	accounts := map[string]cache.AccessTarget{}

	// a map of access rule IDs that match each account ID
	// Prod (123456789012) -> {"rul_123": true}
	accessRulesForAccount := map[string]map[string]bool{}

	for _, rule := range existingRules {
		for _, t := range rule.Targets {
			if t.Type == "accountId" {
				if _, ok := accessRulesForAccount[t.Value]; !ok {
					accessRulesForAccount[t.Value] = map[string]bool{}
				}
				accounts[t.Value] = t
				accessRulesForAccount[t.Value][rule.ID] = true
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
	selectedAccountID := opts.account

	if selectedAccountID == "" {
		clio.Debugw("prompting for accounts", "accounts", accounts)

		prompt := &survey.Select{
			Message: "Account",
			Options: accountOptions,
		}
		err = survey.AskOne(prompt, &selectedAccountOption)
		if err != nil {
			return err
		}

		selectedAccountID = selectedAccountMap[selectedAccountOption]
	}

	selectedAccountInfo, ok := accounts[selectedAccountID]
	if !ok {
		return clierr.New(fmt.Sprintf("account %s not found", selectedAccountID), clierr.Info("run 'granted exp request aws' to see a list of available accounts"))
	}

	ruleIDs := accessRulesForAccount[selectedAccountID]

	// note: we use a map here to de-duplicate accounts.
	// this means that the RuleID in the accounts map is not necessarily
	// the *only* Access Rule which grants access to that account.
	permissionSets := map[string]cache.AccessTarget{}

	for _, rule := range existingRules {
		if _, ok := ruleIDs[rule.ID]; !ok {
			continue
		}

		for _, t := range rule.Targets {
			if t.Type != "permissionSetArn" {
				continue
			}

			permissionSets[t.Value] = t
		}
	}

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

	selectedRole := opts.role

	if selectedRole == "" {
		prompt := &survey.Select{
			Message: "Role",
			Options: permissionSetOptions,
		}
		err = survey.AskOne(prompt, &selectedRole)
		if err != nil {
			return err
		}
	}

	permissionSetArn, ok := permissionSetValues[selectedRole]
	if !ok {
		return clierr.New(fmt.Sprintf("role %s not found", selectedAccountID), clierr.Infof("run 'granted exp request aws --account %s' to see a list of available roles", selectedAccountID))
	}

	selectedPermissionSetRuleIDs := permissionSetRuleIDs[selectedRole]

	// find Access Rules that match the permission set and the account
	// we need to find the intersection between permissionSetRuleIDs and accessRulesForAccount
	// matchingAccessRule tracks the current Access Rule which we'll use to request access against.
	var matchingAccessRule *cache.AccessRule

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
			if matchingAccessRule.RequiresApproval && !rule.RequiresApproval {
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

	reason := opts.reason

	fr, err := frecency.Load("reasons")
	if err != nil {
		return err
	}

	if reason == "" {
		var suggestions []string
		for _, entry := range fr.Entries {
			e := entry.Entry.(string)
			suggestions = append(suggestions, e)
		}

		reasonPrompt := &survey.Input{
			Message: "Reason for access:",
			Help:    "Will be stored in audit trails and associated with you",
			Suggest: func(toComplete string) []string {
				var matched []string
				for _, s := range suggestions {
					if fuzzy.Match(toComplete, s) {
						matched = append(matched, s)
					}
				}

				return matched
			},
		}
		err = survey.AskOne(reasonPrompt, &reason)
		if err != nil {
			return err
		}
	}

	err = fr.Upsert(reason)
	if err != nil {
		clio.Errorw("error updating frecency log", "error", err)
	}

	// only print the one-liner if --reason wasn't provided
	if opts.reason == "" {
		clio.NewLine()
		clio.Infof("Run this one-liner command to request access in future:\ngranted exp request aws --account %s --role %s --reason \"%s\"", selectedAccountID, selectedRole, reason)
		clio.NewLine()
	}

	si := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	si.Suffix = " requesting access..."
	si.Writer = os.Stderr
	si.Start()

	// the current version of the API requires `With` fields to be provided
	// *only* if the Access Rule has multiple options for that field.
	var with []types.CreateRequestWith
	request := types.CreateRequestWith{
		AdditionalProperties: make(map[string][]string),
	}

	var accountIdCount, permissionSetCount int

	for _, t := range matchingAccessRule.Targets {
		if t.Type == "accountId" {
			accountIdCount++
		}
		if t.Type == "permissionSetArn" {
			permissionSetCount++
		}
	}

	// check if the 'accountId' field needs to be included
	if accountIdCount > 1 {
		request.AdditionalProperties["accountId"] = []string{selectedAccountID}
	}

	// check if the 'permissionSetArn' field needs to be included
	if permissionSetCount > 1 {
		request.AdditionalProperties["permissionSetArn"] = []string{permissionSetArn}
	}

	// withPtr is set to null if the `With` field doesn't contain anything.
	// it is used to avoid API bad request errors.
	var withPtr *[]types.CreateRequestWith
	if len(request.AdditionalProperties) > 0 {
		with = append(with, request)
		withPtr = &with
	}

	_, err = cf.UserCreateRequestWithResponse(ctx, types.UserCreateRequestJSONRequestBody{
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

	// find the latest Access Request
	res, err := cf.UserListRequestsWithResponse(ctx, &types.UserListRequestsParams{})
	if err != nil {
		return err
	}

	latestRequest := res.JSON200.Requests[0]

	clio.Infof("Access Request %s - %s", latestRequest.ID, latestRequest.Status)
	reqURL, err := url.Parse(cfcfg.CurrentOrEmpty().DashboardURL)
	if err != nil {
		return err
	}
	reqURL.Path = path.Join("/requests", latestRequest.ID)
	clio.Infof("URL: %s", reqURL)

	fullName := fmt.Sprintf("%s/%s", selectedAccountInfo.Label, selectedRole)

	if latestRequest.Status == types.RequestStatusAPPROVED {
		durationDescription := durafmt.Parse(time.Duration(matchingAccessRule.DurationSeconds) * time.Second).LimitFirstN(1).String()
		profile, err := cfaws.LoadProfileByAccountIdAndRole(selectedAccountID, selectedRole)
		if err != nil {
clio.Debugw("error while trying to automatically detect if profile is active","error",err)
			clio.Warn("Unable to automatically detect whether this profile is ready, however you can try assuming it now.")
			return nil
		}

		if profile == nil {
			clio.Warn("Unable to automatically detect whether this profile is ready, however you can try assuming it now.")
			return nil
		}
		ssoAssumer := cfaws.AwsSsoAssumer{}
		profile.ProfileType = ssoAssumer.Type()

		clio.Debugf("Auto assuming to %s", profile.Name)
		si := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
		si.Suffix = "waiting for the profile to be ready..."
		si.Writer = os.Stderr
		si.Start()

		// run assume with retry such that even if assume fails due to latency issue in provisioning, user will not have to rerun the command.
		_, err = profile.AssumeTerminal(ctx, cfaws.ConfigOpts{
			ShouldRetryAssuming: aws.Bool(true),
		})
		if err != nil {
clio.Debugw("error while trying to automatically detect if profile is active","error",err)
			clio.Warn("Unable to automatically detect whether this profile is ready, however you can try assuming it now.")
			return nil
		}
		si.Stop()

		clio.Successf("[%s] Access is activated (expires in %s)", fullName, durationDescription)
	}

	clio.NewLine()
	clio.Infof("To use the profile with the AWS CLI, sync your ~/.aws/config by running 'granted sso populate'. Then, run:\nexport AWS_PROFILE=%s", fullName)

	return nil
}

func getCachedAccessRules(depID string) (map[string]cache.AccessRule, error) {
	cacheFolder, err := getCacheFolder(depID)
	if err != nil {
		return nil, err
	}

	files, err := os.ReadDir(cacheFolder)
	if err != nil {
		return nil, errors.Wrap(err, "reading cache folder")
	}

	// map of rule ID to the rule itself
	rules := map[string]cache.AccessRule{}

	for _, f := range files {
		// the name of the file is the rule ID (e.g. `rul_123`)
		ruleBytes, err := os.ReadFile(path.Join(cacheFolder, f.Name()))
		if err != nil {
			return nil, err
		}
		var rule cache.AccessRule
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
	Existing     map[string]cache.AccessRule
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
	}

	// otherwise, update the cache
	row := cache.AccessRule{
		ID:                 r.ID,
		Name:               r.Name,
		DeploymentID:       opts.DeploymentID,
		TargetProviderID:   r.Target.Provider.Id,
		TargetProviderType: r.Target.Provider.Type,
		CreatedAt:          r.CreatedAt.Unix(),
		UpdatedAt:          r.UpdatedAt.Unix(),
		DurationSeconds:    r.TimeConstraints.MaxDurationSeconds,
	}

	// our API doesn't easily expose whether manual approval is required
	// on an Access Rule, so we need to fetch approvers separately.
	approvers, err := opts.CF.UserGetAccessRuleApproversWithResponse(ctx, r.ID)
	if err != nil {
		return err
	}

	if len(approvers.JSON200.Users) > 0 {
		row.RequiresApproval = true
	}

	clio.Debugw("updated requires approval", "rule.id", r.ID, "requires_approval", row.RequiresApproval)

	details, err := opts.CF.UserGetAccessRuleWithResponse(ctx, r.ID)
	if err != nil {
		return err
	}

	for k, v := range details.JSON200.Target.Arguments.AdditionalProperties {
		for _, o := range v.Options {
			t := cache.AccessTarget{
				RuleID: r.ID,
				Type:   k,
				Label:  o.Label,
				Value:  o.Value,
			}

			if o.Description != nil {
				t.Description = *o.Description
			}
			row.Targets = append(row.Targets, t)
		}
	}

	clio.Debugw("updated access targets", "rule.id", r.ID, "targets.count", len(row.Targets))

	cacheFolder, err := getCacheFolder(opts.DeploymentID)
	if err != nil {
		return err
	}

	filename := filepath.Join(cacheFolder, r.ID)

	ruleBytes, err := json.Marshal(row)
	if err != nil {
		return err
	}

	err = os.WriteFile(filename, ruleBytes, 0644)
	if err != nil {
		return err
	}

	return nil
}

func getCacheFolder(depID string) (string, error) {
	configFolder, err := config.GrantedConfigFolder()
	if err != nil {
		return "", err
	}
	depURL, err := url.Parse(depID)
	if err != nil {
		return "", err
	}

	// ~/.granted/common-fate-cache/commonfate.example.com/access-rules
	cacheFolder := path.Join(configFolder, "common-fate-cache", depURL.Hostname(), "access-rules")

	if _, err := os.Stat(cacheFolder); os.IsNotExist(err) {
		clio.Debugw("cache folder does not exist, creating", "folder", cacheFolder, "error", err)
		err = os.MkdirAll(cacheFolder, 0755)
		if err != nil {
			return "", errors.Wrapf(err, "creating cache folder %s", cacheFolder)
		}
	}

	return cacheFolder, nil
}

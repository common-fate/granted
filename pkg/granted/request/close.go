package request

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/common-fate/clio"
	"github.com/common-fate/grab"
	"github.com/common-fate/granted/pkg/cfaws"
	"github.com/common-fate/granted/pkg/cfcfg"
	"github.com/common-fate/sdk/eid"
	accessv1alpha1 "github.com/common-fate/sdk/gen/commonfate/access/v1alpha1"
	"github.com/common-fate/sdk/service/access/grants"
	"github.com/common-fate/sdk/service/access/request"
	identitysvc "github.com/common-fate/sdk/service/identity"
	"github.com/urfave/cli/v2"
)

var closeCommand = cli.Command{
	Name:  "close",
	Usage: "Close an active Just-In-Time access to a particular entitlement",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "aws-profile", Required: true, Usage: "Close a JIT access for a particular AWS profile"},
	},
	Action: func(c *cli.Context) error {
		profiles, err := cfaws.LoadProfiles()
		if err != nil {
			return err
		}

		profileName := c.String("aws-profile")

		id := c.String("id")
		clio.Debugw("found active grant matching the profile, attempting to force close grant", "grant", id)

		profile, err := profiles.LoadInitialisedProfile(c.Context, profileName)
		if err != nil {
			return err
		}

		cfg, err := cfcfg.Load(c.Context, profile)
		if err != nil {
			return fmt.Errorf("failed to load cfconfig, cannot check for active grants, %w", err)
		}

		grantsClient := grants.NewFromConfig(cfg)
		idClient := identitysvc.NewFromConfig(cfg)
		callerID, err := idClient.GetCallerIdentity(c.Context, connect.NewRequest(&accessv1alpha1.GetCallerIdentityRequest{}))
		if err != nil {
			return err
		}
		target := eid.New("AWS::Account", profile.AWSConfig.SSOAccountID)

		grants, err := grab.AllPages(c.Context, func(ctx context.Context, nextToken *string) ([]*accessv1alpha1.Grant, *string, error) {
			grants, err := grantsClient.QueryGrants(c.Context, connect.NewRequest(&accessv1alpha1.QueryGrantsRequest{
				Principal: callerID.Msg.Principal.Eid,
				Target:    target.ToAPI(),
				// This API needs to be updated to use specifiers, for now, fetch all active grants and check for a match on the role name
				// Role:      eid.New("AWS::Account", profile.AWSConfig.SSOAccountID).ToAPI(),
				Status: accessv1alpha1.GrantStatus_GRANT_STATUS_ACTIVE.Enum(),
			}))
			if err != nil {
				return nil, nil, err
			}
			return grants.Msg.Grants, &grants.Msg.NextPageToken, nil
		})

		if err != nil {
			clearCacheProfileIfExists(profileName)
			return fmt.Errorf("failed to query for active grants: %w", err)
		}

		accessClient := request.NewFromConfig(cfg)

		for _, grant := range grants {
			if grant.Role.Name == profile.AWSConfig.SSORoleName {
				clio.Debugw("found active grant matching the profile, attempting to close grant", "grant", grant)

				res, err := accessClient.CloseAccessRequest(c.Context, connect.NewRequest(&accessv1alpha1.CloseAccessRequestRequest{
					Id: grant.AccessRequestId,
				}))
				clio.Debugw("result", "res", res)
				if err != nil {
					return err
				}
				clio.Successf("access to target %s and role %s is now closed", target, profile.AWSConfig.SSORoleName)
				return nil
			}
		}

		return fmt.Errorf("no active Access Request found for target %s and role %s", target, profile.AWSConfig.SSORoleName)
	},
}

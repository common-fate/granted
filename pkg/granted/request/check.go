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
	identitysvc "github.com/common-fate/sdk/service/identity"
	"github.com/urfave/cli/v2"
)

var checkCommand = cli.Command{
	Name:  "check",
	Usage: "Check the Common Fate JIT backend to see whether Just-In-Time access to a particular entitlement is active",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "aws-profile", Required: true, Usage: "Check for access for a particular AWS profile"},
	},
	Action: func(c *cli.Context) error {
		profiles, err := cfaws.LoadProfiles()
		if err != nil {
			return err
		}

		profileName := c.String("aws-profile")

		profile, err := profiles.LoadInitialisedProfile(c.Context, profileName)
		if err != nil {
			return err
		}

		cfg, cfConfigErr := cfcfg.Load(c.Context, profile)
		if cfConfigErr != nil {
			clio.Debugw("failed to load cfconfig, skipping check for active grants in a common fate deployment", "error", cfConfigErr)
			return err
		}

		grantsClient := grants.NewFromConfig(cfg)
		idClient := identitysvc.NewFromConfig(cfg)
		callerID, callerIDErr := idClient.GetCallerIdentity(c.Context, connect.NewRequest(&accessv1alpha1.GetCallerIdentityRequest{}))
		if callerIDErr != nil {
			return err
		}
		target := eid.New("AWS::Account", profile.AWSConfig.SSOAccountID)

		grants, queryGrantsErr := grab.AllPages(c.Context, func(ctx context.Context, nextToken *string) ([]*accessv1alpha1.Grant, *string, error) {
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

		if queryGrantsErr != nil {
			clio.Debugw("failed to query for active grants", "error", queryGrantsErr)
			// return the original error
			return err
		}

		for _, grant := range grants {
			if grant.Role.Name == profile.AWSConfig.SSORoleName {
				clio.Debugw("found active grant matching the profile, will retry assuming role", "grant", grant)
				clio.Successf("access to target %s and role %s is currently active", target, profile.AWSConfig.SSORoleName)
				fmt.Println(grant.AccessRequestId)
				return nil
			}
		}
		return fmt.Errorf("no active Access Request found for target %s and role %s", target, profile.AWSConfig.SSORoleName)
	},
}

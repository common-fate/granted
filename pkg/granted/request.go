package granted

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"time"

	"connectrpc.com/connect"
	"github.com/AlecAivazis/survey/v2"
	"github.com/briandowns/spinner"
	"github.com/fatih/color"

	accesscmd "github.com/common-fate/cli/cmd/cli/command/access"
	"github.com/common-fate/cli/printdiags"
	"github.com/common-fate/clio"
	"github.com/urfave/cli/v2"

	"github.com/common-fate/sdk/eid"
	accessv1alpha1 "github.com/common-fate/sdk/gen/commonfate/access/v1alpha1"
	entityv1alpha1 "github.com/common-fate/sdk/gen/commonfate/entity/v1alpha1"
	"github.com/common-fate/sdk/service/access"

	sdkconfig "github.com/common-fate/sdk/config"
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
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "account", Usage: "The AWS account name or ID", Required: true},
		&cli.StringFlag{Name: "role", Usage: "The AWS role", Required: true},
		&cli.StringFlag{Name: "reason", Usage: "A reason for access"},
		&cli.BoolFlag{Name: "confirm", Aliases: []string{"y"}, Usage: "Request access immediately without prompting"},
		&cli.DurationFlag{Name: "duration", Usage: "Duration of request, defaults to max duration of the access rule."},
	},
	Action: func(c *cli.Context) error {
		return requestAccess(c.Context, requestAccessOpts{
			account:  c.String("account"),
			role:     c.String("role"),
			reason:   c.String("reason"),
			duration: c.Duration("duration"),
		})
	},
}

type requestAccessOpts struct {
	account  string
	role     string
	reason   string
	confirm  bool
	duration time.Duration
}

func requestAccess(ctx context.Context, opts requestAccessOpts) error {
	cfg, err := sdkconfig.LoadDefault(ctx)
	if err != nil {
		return err
	}

	apiURL, err := url.Parse(cfg.APIURL)
	if err != nil {
		return err
	}

	accessclient := access.NewFromConfig(cfg)

	availabilities, err := accessclient.QueryAvailabilities(ctx, connect.NewRequest(&accessv1alpha1.QueryAvailabilitiesRequest{}))
	if err != nil {
		return err
	}
}

var Request = cli.Command{
	Name:  "request",
	Usage: "Request access to an entitlement",
	Action: func(c *cli.Context) error {
		ctx := c.Context

		cfg, err := sdkconfig.LoadDefault(ctx)
		if err != nil {
			return err
		}

		apiURL, err := url.Parse(cfg.APIURL)
		if err != nil {
			return err
		}

		accessclient := access.NewFromConfig(cfg)

		availabilities, err := accessclient.QueryAvailabilities(ctx, connect.NewRequest(&accessv1alpha1.QueryAvailabilitiesRequest{}))
		if err != nil {
			return err
		}

		targetsToRoles := map[*entityv1alpha1.EID][]string{}
		var targetLabels []string
		roles := map[string]*entityv1alpha1.EID{}
		targets := map[string]*entityv1alpha1.EID{}

		for _, av := range availabilities.Msg.Availabilities {
			label := fmt.Sprintf("%s (%s::%s)", av.Target.Name, av.Target.Eid.Type, av.Target.Eid.Id)
			targets[label] = av.Target.Eid
			roleLabel := fmt.Sprintf("%s (%s::%s)", av.Role.Name, av.Role.Eid.Type, av.Role.Eid.Id)

			targetsToRoles[av.Target.Eid] = append(targetsToRoles[av.Target.Eid], roleLabel)

			roles[roleLabel] = av.Role.Eid
		}

		for label := range targets {
			targetLabels = append(targetLabels, label)
		}

		var chosenTargetLabel string

		err = survey.AskOne(&survey.Select{
			Message: "Select a target",
			Options: targetLabels,
		}, &chosenTargetLabel)
		if err != nil {
			return err
		}

		selectedTarget := targets[chosenTargetLabel]

		rolelabels := targetsToRoles[selectedTarget]

		var chosenRoleLabel string
		err = survey.AskOne(&survey.Select{
			Message: "Select a role",
			Options: rolelabels,
		}, &chosenRoleLabel)
		if err != nil {
			return err
		}

		selectedRole := roles[chosenRoleLabel]

		var reason string
		err = survey.AskOne(&survey.Input{
			Message: "Why do you need access?",
		}, &reason)
		if err != nil {
			return err
		}

		clio.Debug("ensuring access using Common Fate")

		req := accessv1alpha1.BatchEnsureRequest{
			Entitlements: []*accessv1alpha1.EntitlementInput{
				{
					Target: &accessv1alpha1.Specifier{
						Specify: &accessv1alpha1.Specifier_Eid{
							Eid: selectedTarget,
						},
					},
					Role: &accessv1alpha1.Specifier{
						Specify: &accessv1alpha1.Specifier_Eid{
							Eid: selectedRole,
						},
					},
				},
			},
			Justification: &accessv1alpha1.Justification{
				Reason: &reason,
			},
		}

		// run the dry-run first
		hasChanges, err := accesscmd.DryRun(ctx, apiURL, accessclient, &req, false)
		if err != nil {
			return err
		}
		if !hasChanges {
			fmt.Println("no access changes")

			clio.Info("Use the AWS CLI by running 'assume web-application-prod'")
			clio.Info("Open the AWS Console by running 'assume web-application-prod -c'")
			return nil
		}

		if err != nil {
			return err
		}

		// if we get here, dry-run has passed the user has confirmed they want to proceed.
		req.DryRun = false

		si := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
		si.Suffix = " ensuring access..."
		si.Writer = os.Stderr
		si.Start()

		res, err := accessclient.BatchEnsure(ctx, connect.NewRequest(&req))
		if err != nil {

			si.Stop()
			return err
		}

		//prints response diag messages
		printdiags.Print(res.Msg.Diagnostics, nil)

		si.Stop()

		clio.Debugw("BatchEnsure response", "response", res)

		names := map[eid.EID]string{}

		for _, g := range res.Msg.Grants {
			names[eid.New("Access::Grant", g.Grant.Id)] = g.Grant.Name

			exp := "<invalid expiry>"

			if g.Grant.ExpiresAt != nil {
				exp = accesscmd.ShortDur(time.Until(g.Grant.ExpiresAt.AsTime()))
			}

			switch g.Change {
			case accessv1alpha1.GrantChange_GRANT_CHANGE_ACTIVATED:
				color.New(color.BgHiGreen).Printf("[ACTIVATED]")
				color.New(color.FgGreen).Printf(" %s was activated for %s: %s\n", g.Grant.Name, exp, requestURL(apiURL, g.Grant))

				clio.Info("Use the AWS CLI by running 'assume web-application-prod'")
				clio.Info("Open the AWS Console by running 'assume web-application-prod -c'")

				continue

			case accessv1alpha1.GrantChange_GRANT_CHANGE_EXTENDED:
				color.New(color.BgBlue).Printf("[EXTENDED]")
				color.New(color.FgBlue).Printf(" %s was extended for another %s: %s\n", g.Grant.Name, exp, requestURL(apiURL, g.Grant))
				continue

			case accessv1alpha1.GrantChange_GRANT_CHANGE_REQUESTED:
				color.New(color.BgHiYellow, color.FgBlack).Printf("[REQUESTED]")
				color.New(color.FgYellow).Printf(" %s requires approval: %s\n", g.Grant.Name, requestURL(apiURL, g.Grant))
				continue

			case accessv1alpha1.GrantChange_GRANT_CHANGE_PROVISIONING_FAILED:
				// shouldn't happen in the dry-run request but handle anyway
				color.New(color.FgRed).Printf("[ERROR] %s failed provisioning: %s\n", g.Grant.Name, requestURL(apiURL, g.Grant))
				continue
			}

			switch g.Grant.Status {
			case accessv1alpha1.GrantStatus_GRANT_STATUS_ACTIVE:
				color.New(color.FgGreen).Printf("[ACTIVE] %s is already active for the next %s: %s\n", g.Grant.Name, exp, requestURL(apiURL, g.Grant))

				clio.Info("Use the AWS CLI by running 'assume web-application-prod'")
				clio.Info("Open the AWS Console by running 'assume web-application-prod -c'")

				continue

			case accessv1alpha1.GrantStatus_GRANT_STATUS_PENDING:
				color.New(color.FgWhite).Printf("[PENDING] %s is already pending: %s\n", g.Grant.Name, requestURL(apiURL, g.Grant))
				continue
			case accessv1alpha1.GrantStatus_GRANT_STATUS_CLOSED:
				color.New(color.FgWhite).Printf("[CLOSED] %s is closed but was still returned: %s\n. This is most likely due to an error in Common Fate and should be reported to our team: support@commonfate.io.", g.Grant.Name, requestURL(apiURL, g.Grant))
				continue
			}

			color.New(color.FgWhite).Printf("[UNSPECIFIED] %s is in an unspecified status: %s\n. This is most likely due to an error in Common Fate and should be reported to our team: support@commonfate.io.", g.Grant.Name, requestURL(apiURL, g.Grant))
		}

		printdiags.Print(res.Msg.Diagnostics, names)

		return nil
	},
}

func requestURL(apiURL *url.URL, grant *accessv1alpha1.Grant) string {
	p := apiURL.JoinPath("access", "requests", grant.AccessRequestId)
	return p.String()
}

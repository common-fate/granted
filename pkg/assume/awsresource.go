package assume

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/common-fate/clio"
	"github.com/common-fate/grab"
	"github.com/common-fate/granted/pkg/cfaws"
	"github.com/common-fate/granted/pkg/hook/accessrequesthook"
	cfflags "github.com/common-fate/granted/pkg/urfav_overrides"
	"github.com/common-fate/sdk/config"
	accessv1alpha1 "github.com/common-fate/sdk/gen/commonfate/access/v1alpha1"
	"github.com/common-fate/sdk/service/access"
	"github.com/common-fate/sdk/service/access/grants"
	"github.com/mattn/go-runewidth"
	sethRetry "github.com/sethvargo/go-retry"
	"github.com/urfave/cli/v2"
	"google.golang.org/protobuf/types/known/durationpb"
)

func ResourceAccess(cliCtx *cli.Context) (*cfaws.Profile, error) {
	ctx := cliCtx.Context
	assumeFlags, err := cfflags.New("assumeFlags", GlobalFlags(), cliCtx)
	if err != nil {
		return nil, err
	}

	cfg, err := config.LoadDefault(ctx)
	if err != nil {
		return nil, err
	}

	err = cfg.Initialize(ctx, config.InitializeOpts{})
	if err != nil {
		return nil, err
	}

	client := access.NewFromConfig(cfg)

	entitlements, err := grab.AllPages(ctx, func(ctx context.Context, nextToken *string) ([]*accessv1alpha1.Entitlement, *string, error) {
		res, err := client.QueryEntitlements(ctx, connect.NewRequest(&accessv1alpha1.QueryEntitlementsRequest{
			PageToken:  grab.Value(nextToken),
			TargetType: grab.Ptr("AWS::S3::Bucket"),
		}))
		if err != nil {
			return nil, nil, err
		}
		return res.Msg.Entitlements, &res.Msg.NextPageToken, nil
	})
	if err != nil {
		return nil, err
	}

	type Column struct {
		Title string
		Width int
	}
	cols := []Column{{Title: "S3 Bucket", Width: 60}, {Title: "Role", Width: 40}}
	var s = make([]string, 0, len(cols))
	for _, col := range cols {
		style := lipgloss.NewStyle().Width(col.Width).MaxWidth(col.Width).Inline(true)
		renderedCell := style.Render(runewidth.Truncate(col.Title, col.Width, "…"))
		s = append(s, lipgloss.NewStyle().Bold(true).Padding(0).Render(renderedCell))
	}
	header := lipgloss.NewStyle().PaddingLeft(6).Render(lipgloss.JoinHorizontal(lipgloss.Left, s...))
	var options []huh.Option[*accessv1alpha1.Entitlement]

	for _, entitlement := range entitlements {
		style := lipgloss.NewStyle().Width(cols[0].Width).MaxWidth(cols[0].Width).Inline(true)
		target := lipgloss.NewStyle().Padding(0).Render(style.Render(runewidth.Truncate(entitlement.Target.Display(), cols[0].Width, "…")))

		style = lipgloss.NewStyle().Width(cols[1].Width).MaxWidth(cols[1].Width).Inline(true)
		role := lipgloss.NewStyle().Padding(0).Render(style.Render(runewidth.Truncate(entitlement.Role.Display(), cols[1].Width, "…")))

		options = append(options, huh.Option[*accessv1alpha1.Entitlement]{
			Key:   lipgloss.JoinHorizontal(lipgloss.Left, target, role),
			Value: entitlement,
		})
	}

	selector := huh.NewMultiSelect[*accessv1alpha1.Entitlement]().
		Options(options...).
		Title("Select the resources you need access to:").
		Description(header).WithTheme(huh.ThemeBase())
	err = selector.Run()
	if err != nil {
		return nil, err
	}

	selected := selector.GetValue().([]*accessv1alpha1.Entitlement)

	// target = entitlement.Target.Eid.Display()
	// role = entitlement.Role.Eid.Display()

	clio.Infow("selected", "value", selected)

	var duration *durationpb.Duration
	if assumeFlags.String("duration") != "" {
		d, err := time.ParseDuration(assumeFlags.String("duration"))
		if err != nil {
			return nil, err
		}
		duration = durationpb.New(d)
	}

	input := accessrequesthook.NoEntitlementAccessInput{
		Entitlements: grab.Map(selected, func(t *accessv1alpha1.Entitlement) *accessv1alpha1.EntitlementInput {
			return &accessv1alpha1.EntitlementInput{
				Target: &accessv1alpha1.Specifier{
					Specify: &accessv1alpha1.Specifier_Eid{
						Eid: t.Target.Eid,
					},
				},
				Role: &accessv1alpha1.Specifier{
					Specify: &accessv1alpha1.Specifier_Eid{
						Eid: t.Role.Eid,
					},
				},
			}
		}),
		Reason:    assumeFlags.String("reason"),
		Duration:  duration,
		Confirm:   assumeFlags.Bool("confirm"),
		Wait:      assumeFlags.Bool("wait"),
		StartTime: time.Now(),
	}
	hook := accessrequesthook.Hook{}
	retry, result, err := hook.NoEntitlementAccess(ctx, cfg, input)
	if err != nil {
		return nil, err
	}

	retryDuration := time.Minute * 1
	if assumeFlags.Bool("wait") {
		//if wait is specified, increase the timeout to 15 minutes.
		retryDuration = time.Minute * 15
	}

	if retry {
		// reset the start time for the timer (otherwise it shows 2s, 7s, 12s etc)
		input.StartTime = time.Now()

		b := sethRetry.NewConstant(5 * time.Second)
		b = sethRetry.WithMaxDuration(retryDuration, b)
		err = sethRetry.Do(ctx, b, func(ctx context.Context) (err error) {

			//also proactively check if request has been approved and attempt to activate
			result, err = hook.RetryNoEntitlementAccess(ctx, cfg, input)
			if err != nil {

				return sethRetry.RetryableError(err)
			}

			return nil
		})
		if err != nil {
			return nil, err
		}

	}

	clio.Info("Grant is activated")

	if result == nil || len(result.Grants) == 0 {
		return nil, errors.New("could not load grant from Common Fate")
	}

	firstGrant := result.Grants[0]

	grantsClient := grants.NewFromConfig(cfg)

	res, err := grantsClient.GetGrantOutput(ctx, connect.NewRequest(&accessv1alpha1.GetGrantOutputRequest{
		Id: firstGrant.Grant.Id,
	}))
	if err != nil {
		return nil, err
	}
	dynRole := res.Msg.GetOutputAwsDynamicRole()
	if dynRole == nil {
		return nil, errors.New("expected grant output to be dynamic role")
	}

	p := &cfaws.Profile{
		Name:        dynRole.Id,
		ProfileType: "AWS_SSO",
		AWSConfig: awsConfig.SharedConfig{
			SSOAccountID: dynRole.AccountId,
			SSORoleName:  dynRole.SsoRoleName,
			SSORegion:    dynRole.SsoRegion,
			SSOStartURL:  dynRole.SsoStartUrl,
		},
		Initialised: true,
	}

	return p, nil
}

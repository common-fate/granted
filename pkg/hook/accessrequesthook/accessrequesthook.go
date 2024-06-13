package accessrequesthook

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/AlecAivazis/survey/v2"
	"github.com/briandowns/spinner"
	accesscmd "github.com/common-fate/cli/cmd/cli/command/access"
	"github.com/common-fate/cli/printdiags"
	"github.com/common-fate/clio"
	"github.com/common-fate/granted/pkg/cfaws"
	"github.com/common-fate/granted/pkg/cfcfg"
	"github.com/common-fate/sdk/eid"
	accessv1alpha1 "github.com/common-fate/sdk/gen/commonfate/access/v1alpha1"
	"github.com/common-fate/sdk/gen/commonfate/access/v1alpha1/accessv1alpha1connect"
	"github.com/common-fate/sdk/loginflow"
	"github.com/common-fate/sdk/service/access"
	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
	"google.golang.org/protobuf/encoding/protojson"
	durationpb "google.golang.org/protobuf/types/known/durationpb"
)

type Hook struct{}

type NoAccessInput struct {
	Profile   *cfaws.Profile
	Reason    string
	Duration  *durationpb.Duration
	Confirm   bool
	Wait      bool
	StartTime time.Time
}

func (h Hook) NoAccess(ctx context.Context, input NoAccessInput) (retry bool, err error) {
	cfg, err := cfcfg.Load(ctx, input.Profile)
	if err != nil {
		return false, err
	}

	target := eid.New("AWS::Account", input.Profile.AWSConfig.SSOAccountID)
	role := input.Profile.AWSConfig.SSORoleName

	clio.Infof("You don't currently have access to %s, checking if we can request access...\t[target=%s, role=%s, url=%s]", input.Profile.Name, target, role, cfg.AccessURL)

	apiURL, err := url.Parse(cfg.APIURL)
	if err != nil {
		return false, err
	}

	accessclient := access.NewFromConfig(cfg)

	req := accessv1alpha1.BatchEnsureRequest{
		Entitlements: []*accessv1alpha1.EntitlementInput{
			{
				Target: &accessv1alpha1.Specifier{
					Specify: &accessv1alpha1.Specifier_Eid{
						Eid: target.ToAPI(),
					},
				},
				Role: &accessv1alpha1.Specifier{
					Specify: &accessv1alpha1.Specifier_Lookup{
						Lookup: role,
					},
				},
				Duration: input.Duration,
			},
		},
		Justification: &accessv1alpha1.Justification{},
	}

	hasChanges, validation, err := DryRun(ctx, apiURL, accessclient, &req, false, input.Confirm)
	if shouldRefreshLogin(err) {
		clio.Debugw("prompting user login because token is expired", "error_details", err.Error())
		// NOTE(chrnorm): ideally we'll bubble up a more strongly typed error in future here, to avoid the string comparison on the error message.

		// the OAuth2.0 token is expired so we should prompt the user to log in
		clio.Infof("You need to log in to Common Fate")

		lf := loginflow.NewFromConfig(cfg)
		err = lf.Login(ctx)
		if err != nil {
			return false, err
		}

		accessclient = access.NewFromConfig(cfg)

		// retry the Dry Run again
		hasChanges, validation, err = DryRun(ctx, apiURL, accessclient, &req, false, input.Confirm)
	}

	if err != nil {
		return false, err
	}
	if !hasChanges {
		// shouldn't retry assuming if there aren't any proposed access changes
		return false, errors.New("no access changes")
	}

	// if we get here, dry-run has passed the user has confirmed they want to proceed.
	req.DryRun = false

	if input.Reason != "" {
		req.Justification.Reason = &input.Reason
	} else {
		if validation != nil && validation.HasReason {
			if !IsTerminal(os.Stdin.Fd()) {
				return false, errors.New("detected a noninteractive terminal: a reason is required to make this access request, to apply the planned changes please re-run with the --reason flag")
			}

			var customReason string
			msg := "Reason for access (Required)"
			reasonPrompt := &survey.Input{
				Message: msg,
				Help:    "Will be stored in audit trails and associated with your request",
			}
			withStdio := survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)
			err = survey.AskOne(reasonPrompt, &customReason, withStdio, survey.WithValidator(survey.Required))

			if err != nil {
				return false, err
			}

			req.Justification.Reason = &customReason
		}
	}
	// the spinner must be started after prompting for reason, otherwise the prompt gets hidden
	si := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	si.Suffix = " ensuring access..."
	si.Writer = os.Stderr
	si.Start()

	res, err := accessclient.BatchEnsure(ctx, connect.NewRequest(&req))
	if err != nil {
		si.Stop()
		return false, err
	}
	si.Stop()
	//prints response diag messages
	printdiags.Print(res.Msg.Diagnostics, nil)

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
			color.New(color.BgHiGreen).Fprintf(os.Stderr, "[ACTIVATED]")
			color.New(color.FgGreen).Fprintf(os.Stderr, " %s was activated for %s: %s\n", g.Grant.Name, exp, requestURL(apiURL, g.Grant))

			retry = true

			continue

		case accessv1alpha1.GrantChange_GRANT_CHANGE_EXTENDED:
			color.New(color.BgBlue).Fprintf(os.Stderr, "[EXTENDED]")
			color.New(color.FgBlue).Fprintf(os.Stderr, " %s was extended for another %s: %s\n", g.Grant.Name, exp, requestURL(apiURL, g.Grant))

			retry = true

			continue

		case accessv1alpha1.GrantChange_GRANT_CHANGE_REQUESTED:
			color.New(color.BgHiYellow, color.FgBlack).Fprintf(os.Stderr, "[REQUESTED]")
			color.New(color.FgYellow).Fprintf(os.Stderr, " %s requires approval: %s\n", g.Grant.Name, requestURL(apiURL, g.Grant))

			if input.Wait {
				return true, nil
			}

			return false, errors.New("applying access was attempted but the resources requested require approval before activation")

		case accessv1alpha1.GrantChange_GRANT_CHANGE_PROVISIONING_FAILED:
			// shouldn't happen in the dry-run request but handle anyway
			color.New(color.FgRed).Fprintf(os.Stderr, "[ERROR] %s failed provisioning: %s\n", g.Grant.Name, requestURL(apiURL, g.Grant))

			return false, errors.New("access provisioning failed")
		}

		switch g.Grant.Status {
		case accessv1alpha1.GrantStatus_GRANT_STATUS_ACTIVE:
			color.New(color.FgGreen).Fprintf(os.Stderr, "[ACTIVE] %s is already active for the next %s: %s\n", g.Grant.Name, exp, requestURL(apiURL, g.Grant))

			retry = true

			continue

		case accessv1alpha1.GrantStatus_GRANT_STATUS_PENDING:
			color.New(color.FgWhite).Fprintf(os.Stderr, "[PENDING] %s is already pending: %s\n", g.Grant.Name, requestURL(apiURL, g.Grant))

			return false, errors.New("access is pending approval")

		case accessv1alpha1.GrantStatus_GRANT_STATUS_CLOSED:
			color.New(color.FgWhite).Fprintf(os.Stderr, "[CLOSED] %s is closed but was still returned: %s\n. This is most likely due to an error in Common Fate and should be reported to our team: support@commonfate.io.", g.Grant.Name, requestURL(apiURL, g.Grant))

			return false, errors.New("grant was closed")

		default:
			color.New(color.FgWhite).Fprintf(os.Stderr, "[UNSPECIFIED] %s is in an unspecified status: %s\n. This is most likely due to an error in Common Fate and should be reported to our team: support@commonfate.io.", g.Grant.Name, requestURL(apiURL, g.Grant))
			return false, errors.New("grant was in an unspecified state")
		}

	}

	printdiags.Print(res.Msg.Diagnostics, names)

	return retry, nil

}

func (h Hook) RetryAccess(ctx context.Context, input NoAccessInput) error {
	cfg, err := cfcfg.Load(ctx, input.Profile)
	if err != nil {
		return err
	}

	accessclient := access.NewFromConfig(cfg)
	target := eid.New("AWS::Account", input.Profile.AWSConfig.SSOAccountID)
	role := input.Profile.AWSConfig.SSORoleName
	req := accessv1alpha1.BatchEnsureRequest{
		Entitlements: []*accessv1alpha1.EntitlementInput{
			{
				Target: &accessv1alpha1.Specifier{
					Specify: &accessv1alpha1.Specifier_Eid{
						Eid: target.ToAPI(),
					},
				},
				Role: &accessv1alpha1.Specifier{
					Specify: &accessv1alpha1.Specifier_Lookup{
						Lookup: role,
					},
				},
				Duration: input.Duration,
			},
		},
		Justification: &accessv1alpha1.Justification{},
	}

	res, err := accessclient.BatchEnsure(ctx, connect.NewRequest(&req))
	if err != nil {
		return err
	}

	now := time.Now()
	elapsed := now.Sub(input.StartTime).Round(time.Second * 10)

	for _, g := range res.Msg.Grants {

		// if grant is approved but the change is unspecified then the user is not able to automatically activate
		if g.Grant.Approved && g.Change == accessv1alpha1.GrantChange_GRANT_CHANGE_UNSPECIFIED {
			clio.Infof("Request was approved but failed to activate, you might not have permission to activate. You can try and activate the access using the Common Fate web console. [%s elapsed]", elapsed)
		}

		if !g.Grant.Approved {
			clio.Infof("Waiting for request to be approved... [%s elapsed]", elapsed)
		}

	}
	return nil
}

func requestURL(apiURL *url.URL, grant *accessv1alpha1.Grant) string {
	p := apiURL.JoinPath("access", "requests", grant.AccessRequestId)
	return p.String()
}

func DryRun(ctx context.Context, apiURL *url.URL, client accessv1alpha1connect.AccessServiceClient, req *accessv1alpha1.BatchEnsureRequest, jsonOutput bool, confirm bool) (bool, *accessv1alpha1.Validation, error) {
	req.DryRun = true

	si := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	si.Suffix = " planning access changes..."
	si.Writer = os.Stderr
	si.Start()

	res, err := client.BatchEnsure(ctx, connect.NewRequest(req))
	if err != nil {
		si.Stop()
		return false, nil, err
	}

	si.Stop()

	clio.Debugw("BatchEnsure response", "response", res)

	if jsonOutput {
		resJSON, err := protojson.Marshal(res.Msg)
		if err != nil {
			return false, nil, err
		}
		fmt.Println(string(resJSON))

		return false, nil, errors.New("exiting because --output=json was specified: use --output=text to show an interactive prompt, or use --confirm to proceed with the changes")
	}

	names := map[eid.EID]string{}

	var hasChanges bool

	for _, g := range res.Msg.Grants {
		names[eid.New("Access::Grant", g.Grant.Id)] = g.Grant.Name

		exp := "<invalid expiry>"

		if g.Grant.ExpiresAt != nil {
			exp = ShortDur(time.Until(g.Grant.ExpiresAt.AsTime()))
		}

		if g.Change > 0 {
			hasChanges = true
		}

		switch g.Change {
		case accessv1alpha1.GrantChange_GRANT_CHANGE_ACTIVATED:
			color.New(color.BgHiGreen).Fprintf(os.Stderr, "[WILL ACTIVATE]")
			color.New(color.FgGreen).Fprintf(os.Stderr, " %s will be activated for %s: %s\n", g.Grant.Name, exp, requestURL(apiURL, g.Grant))
			continue

		case accessv1alpha1.GrantChange_GRANT_CHANGE_EXTENDED:
			color.New(color.BgBlue).Fprintf(os.Stderr, "[WILL EXTEND]")
			color.New(color.FgBlue).Fprintf(os.Stderr, " %s will be extended for another %s: %s\n", g.Grant.Name, exp, requestURL(apiURL, g.Grant))
			continue

		case accessv1alpha1.GrantChange_GRANT_CHANGE_REQUESTED:
			color.New(color.BgHiYellow, color.FgBlack).Fprintf(os.Stderr, "[WILL REQUEST]")
			color.New(color.FgYellow).Fprintf(os.Stderr, " %s will require approval\n", g.Grant.Name)
			continue

		case accessv1alpha1.GrantChange_GRANT_CHANGE_PROVISIONING_FAILED:
			// shouldn't happen in the dry-run request but handle anyway
			color.New(color.FgRed).Fprintf(os.Stderr, "[ERROR] %s will fail provisioning\n", g.Grant.Name)
			continue
		}

		switch g.Grant.Status {
		case accessv1alpha1.GrantStatus_GRANT_STATUS_ACTIVE:
			color.New(color.FgGreen).Fprintf(os.Stderr, "[ACTIVE] %s is already active for the next %s: %s\n", g.Grant.Name, exp, requestURL(apiURL, g.Grant))
			continue
		case accessv1alpha1.GrantStatus_GRANT_STATUS_PENDING:
			color.New(color.FgWhite).Fprintf(os.Stderr, "[PENDING] %s is already pending: %s\n", g.Grant.Name, requestURL(apiURL, g.Grant))
			continue
		case accessv1alpha1.GrantStatus_GRANT_STATUS_CLOSED:
			color.New(color.FgWhite).Fprintf(os.Stderr, "[CLOSED] %s is closed but was still returned: %s\n. This is most likely due to an error in Common Fate and should be reported to our team: support@commonfate.io.", g.Grant.Name, requestURL(apiURL, g.Grant))
			continue
		}

		color.New(color.FgWhite).Fprintf(os.Stderr, "[UNSPECIFIED] %s is in an unspecified status: %s\n. This is most likely due to an error in Common Fate and should be reported to our team: support@commonfate.io.", g.Grant.Name, requestURL(apiURL, g.Grant))
	}

	printdiags.Print(res.Msg.Diagnostics, names)

	if !hasChanges {
		return false, nil, nil
	}

	if !confirm {
		if !IsTerminal(os.Stdin.Fd()) {
			return false, nil, errors.New("detected a noninteractive terminal: to apply the planned changes please re-run with the --confirm-access-request flag")
		}

		withStdio := survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)
		confirmPrompt := survey.Confirm{
			Message: "Apply proposed access changes",
		}
		err = survey.AskOne(&confirmPrompt, &confirm, withStdio)
		if err != nil {
			return false, nil, err
		}
	}

	clio.Info("Attempting to grant access...")
	return confirm, res.Msg.Validation, nil
}

func IsTerminal(fd uintptr) bool {
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}

func ShortDur(d time.Duration) string {
	if d > time.Minute {
		d = d.Round(time.Minute)
	} else {
		d = d.Round(time.Second)
	}

	s := d.String()
	if strings.HasSuffix(s, "m0s") {
		s = s[:len(s)-2]
	}
	if strings.HasSuffix(s, "h0m") {
		s = s[:len(s)-2]
	}
	return s
}

func shouldRefreshLogin(err error) bool {
	if err == nil {
		return false
	}
	if strings.Contains(err.Error(), "oauth2: token expired") {
		return true
	}
	if strings.Contains(err.Error(), "oauth2: invalid grant") {
		return true
	}

	return false
}

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
	sdkconfig "github.com/common-fate/sdk/config"
	"github.com/common-fate/sdk/eid"
	accessv1alpha1 "github.com/common-fate/sdk/gen/commonfate/access/v1alpha1"
	"github.com/common-fate/sdk/gen/commonfate/access/v1alpha1/accessv1alpha1connect"
	"github.com/common-fate/sdk/loginflow"
	"github.com/common-fate/sdk/service/access"
	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
	"google.golang.org/protobuf/encoding/protojson"
)

type Hook struct{}

func getCommonFateURL(profile *cfaws.Profile) (*url.URL, error) {
	if profile == nil {
		clio.Debugw("skipping loading Common Fate SDK from URL", "reason", "profile was nil")
		return nil, nil
	}
	if profile.RawConfig == nil {
		clio.Debugw("skipping loading Common Fate SDK from URL", "reason", "profile.RawConfig was nil")
		return nil, nil
	}
	if !profile.RawConfig.HasKey("common_fate_url") {
		clio.Debugw("skipping loading Common Fate SDK from URL", "reason", "profile does not have key common_fate_url", "profile_keys", profile.RawConfig.KeyStrings())
		return nil, nil
	}
	key, err := profile.RawConfig.GetKey("common_fate_url")
	if err != nil {
		return nil, err
	}

	u, err := url.Parse(key.Value())
	if err != nil {
		return nil, fmt.Errorf("invalid common_fate_url (%s): %w", key.Value(), err)
	}

	return u, nil
}

func (h Hook) NoAccess(ctx context.Context, profile *cfaws.Profile) (retry bool, err error) {
	var cfg *sdkconfig.Context

	cfURL, err := getCommonFateURL(profile)
	if err != nil {
		return false, err
	}

	if cfURL != nil {
		cfURL = cfURL.JoinPath("config.json")

		clio.Debugw("configuring Common Fate SDK from URL", "url", cfURL.String())

		cfg, err = sdkconfig.New(ctx, sdkconfig.Opts{
			ConfigSources: []string{cfURL.String()},
		})
		if err != nil {
			return false, err
		}
	} else {
		// if we can't load the Common Fate SDK config (e.g. if `~/.cf/config` is not present)
		// we can't request access through the Common Fate platform.
		cfg, err = sdkconfig.LoadDefault(ctx)
		if err != nil {
			clio.Debugw("error loading Common Fate SDK config", "error", err)
			return false, nil
		}
	}

	target := eid.New("AWS::Account", profile.AWSConfig.SSOAccountID)
	role := profile.AWSConfig.SSORoleName

	clio.Infof("You don't currently have access to %s, checking if we can request access...\t[target=%s, role=%s, url=%s]", profile.Name, target, role, cfg.AccessURL)

	apiURL, err := url.Parse(cfg.APIURL)
	if err != nil {
		return false, err
	}

	accessclient := access.NewFromConfig(cfg)

	reason := "Granted CLI access request for " + profile.Name

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
			},
		},
		Justification: &accessv1alpha1.Justification{
			Reason: &reason,
		},
	}

	hasChanges, err := DryRun(ctx, apiURL, accessclient, &req, false)
	if err != nil && strings.Contains(err.Error(), "oauth2: token expired") {
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
		hasChanges, err = DryRun(ctx, apiURL, accessclient, &req, false)
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

	si := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	si.Suffix = " ensuring access..."
	si.Writer = os.Stderr
	si.Start()

	res, err := accessclient.BatchEnsure(ctx, connect.NewRequest(&req))
	if err != nil {
		si.Stop()
		return false, err
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

			return false, errors.New("access is pending approval")

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

func requestURL(apiURL *url.URL, grant *accessv1alpha1.Grant) string {
	p := apiURL.JoinPath("access", "requests", grant.AccessRequestId)
	return p.String()
}

func DryRun(ctx context.Context, apiURL *url.URL, client accessv1alpha1connect.AccessServiceClient, req *accessv1alpha1.BatchEnsureRequest, jsonOutput bool) (bool, error) {
	req.DryRun = true

	si := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	si.Suffix = " planning access changes..."
	si.Writer = os.Stderr
	si.Start()

	res, err := client.BatchEnsure(ctx, connect.NewRequest(req))
	if err != nil {
		si.Stop()
		return false, err
	}

	si.Stop()

	clio.Debugw("BatchEnsure response", "response", res)

	if jsonOutput {
		resJSON, err := protojson.Marshal(res.Msg)
		if err != nil {
			return false, err
		}
		fmt.Println(string(resJSON))

		return false, errors.New("exiting because --output=json was specified: use --output=text to show an interactive prompt, or use --confirm to proceed with the changes")
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
		return false, nil
	}

	if !IsTerminal(os.Stdin.Fd()) {
		return false, errors.New("detected a noninteractive terminal: to apply the planned changes please re-run with the --confirm-access-request flag")
	}

	withStdio := survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)
	confirm := survey.Confirm{
		Message: "Apply proposed access changes",
	}
	var proceed bool
	err = survey.AskOne(&confirm, &proceed, withStdio)
	if err != nil {
		return false, err
	}
	return proceed, nil
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

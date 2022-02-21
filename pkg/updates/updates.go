package updates

import (
	"time"

	updatev1alpha1 "github.com/common-fate/cf-protos/gen/proto/go/update/v1alpha1"
	"github.com/common-fate/granted/internal/build"
	"github.com/common-fate/granted/pkg/api"
	"github.com/common-fate/granted/pkg/config"
	"github.com/urfave/cli/v2"
)

// Will check once per day for updates
// the last day checked is stored in the local config cache
// this function will fail silently
func Check(c *cli.Context) (string, bool) {
	updateCheckerApiUrl := c.String("update-checker-api-url")
	if updateCheckerApiUrl != "" {
		cfg, err := config.Load()
		if err != nil {
			return "", false
		}
		if cfg.LastCheckForUpdates != time.Now().Weekday() {
			cc, err := api.NewClientConn(c.Context, updateCheckerApiUrl)
			if err != nil {
				return "", false
			}
			updateClient := updatev1alpha1.NewUpdateServiceClient(cc)
			r, err := updateClient.CheckForUpdates(c.Context, &updatev1alpha1.CheckForUpdatesRequest{Version: build.Version, Application: "granted-cli"})
			if err != nil {
				return "", false
			}
			cfg.LastCheckForUpdates = time.Now().Weekday()
			_ = cfg.Save()
			return r.Message, r.UpdateRequired
		}
	}
	return "", false
}

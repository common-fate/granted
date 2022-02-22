package updates

import (
	"fmt"
	"os"
	"sync"
	"time"

	updatev1alpha1 "github.com/common-fate/cf-protos/gen/proto/go/update/v1alpha1"
	"github.com/common-fate/granted/internal/build"
	"github.com/common-fate/granted/pkg/api"
	"github.com/common-fate/granted/pkg/config"
	"github.com/common-fate/granted/pkg/debug"
	"github.com/urfave/cli/v2"
)

// Will check once per day for updates
// the last day checked is stored in the local config cache
// this function will fail silently
func Check(c *cli.Context) (string, bool) {
	updateCheckerApiUrl := c.String("update-checker-api-url")
	if updateCheckerApiUrl != "" {
		debug.Fprintf(debug.VerbosityDebug, os.Stderr, "starting update check")
		cfg, err := config.Load()
		if err != nil {
			return "", false
		}
		if cfg.LastCheckForUpdates != time.Now().Weekday() {
			debug.Fprintf(debug.VerbosityDebug, os.Stderr, "connecting to update checker")
			cc, err := api.NewClientConn(c.Context, updateCheckerApiUrl)
			if err != nil {
				debug.Fprintf(debug.VerbosityDebug, os.Stderr, "failed connecting to update checker: %s", err.Error())
				return "", false
			}
			debug.Fprintf(debug.VerbosityDebug, os.Stderr, "connected to update checker")
			updateClient := updatev1alpha1.NewUpdateServiceClient(cc)
			r, err := updateClient.CheckForUpdates(c.Context, &updatev1alpha1.CheckForUpdatesRequest{Version: build.Version, Application: "granted-cli"})
			if err != nil {
				debug.Fprintf(debug.VerbosityDebug, os.Stderr, "failed checking for updates: %s", err.Error())
				return "", false
			}
			cfg.LastCheckForUpdates = time.Now().Weekday()
			_ = cfg.Save()
			return r.Message, r.UpdateRequired
		}
	}
	return "", false
}

// WithUpdateCheck takes a urfave/cli actionFunc
// and wraps it in "middleware"
// checking for updates in the background if required
func WithUpdateCheck(action cli.ActionFunc) cli.ActionFunc {
	return func(c *cli.Context) error {

		var wg sync.WaitGroup
		wg.Add(1)
		// run the update check async to avoid blocking the users
		var updateAvailable bool
		var msg string
		go func() {
			msg, updateAvailable = Check(c)
			wg.Done()
		}()

		err := action(c)
		wg.Wait()
		if updateAvailable {
			fmt.Fprintf(os.Stderr, "\n%s\n", msg)
		}
		return err
	}
}

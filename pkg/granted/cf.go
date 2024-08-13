package granted

import (
	"github.com/common-fate/clio"
	"github.com/common-fate/granted/pkg/cfaws"
	"github.com/common-fate/granted/pkg/cfcfg"
	sdkconfig "github.com/common-fate/sdk/config"
	"github.com/pkg/browser"
	"github.com/urfave/cli/v2"
)

var CFCommand = cli.Command{
	Name:  "cf",
	Usage: "Open Common Fate in the browser",
	Action: func(c *cli.Context) error {

		ctx := c.Context
		clio.Infof("Opening the Common Fate console in your browser...")
		consoleURL := ""
		cfFileConfig, err := sdkconfig.LoadDefault(ctx)
		if err != nil {
			// fall back to trying to find a common fate url from profiles
			profiles, err := cfaws.LoadProfiles()
			if err != nil {
				return err
			}

			for _, profile := range profiles.ProfileNames {
				p, err := profiles.Profile(profile)
				if err != nil {
					return err
				}
				u, err := cfcfg.GetCommonFateURL(p)
				if err != nil {
					clio.Debug(err)
				} else {
					consoleURL = u.String()
					break
				}
			}
			return err
		} else {
			consoleURL = cfFileConfig.APIURL
		}

		err = browser.OpenURL(consoleURL)
		if err != nil {
			// fail silently
			clio.Debug(err.Error())
		}

		return nil
	},
}

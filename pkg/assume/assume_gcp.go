package assume

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/common-fate/clio"
	"github.com/common-fate/clio/clierr"
	"github.com/common-fate/granted/pkg/cfgcp"
	"github.com/common-fate/granted/pkg/config"
	"github.com/common-fate/granted/pkg/forkprocess"
	"github.com/common-fate/granted/pkg/launcher"
	"github.com/common-fate/granted/pkg/testable"
	cfflags "github.com/common-fate/granted/pkg/urfav_overrides"
	"github.com/urfave/cli/v2"
)

type AssumeGCP struct {
	assumeFlags   *cfflags.Flags
	getConsoleURL bool
}

func (a AssumeGCP) processArgsAndExecFlag(c *cli.Context, assumeFlags *cfflags.Flags) (string, *execConfig, error) {
	//cut off the gcp and use the same code as aws
	slice := c.Args().Slice()[1:]

	if len(slice) == 0 {
		return "", nil, nil
	}

	if len(slice) > 0 && strings.HasPrefix(slice[0], "-") {
		return "", nil, nil
	} else {
		return slice[0], nil, nil
	}

}

func (a AssumeGCP) Assume(ctx *cli.Context) error {
	configName, _, err := a.processArgsAndExecFlag(ctx, a.assumeFlags)
	if err != nil {
		return err
	}

	projectKeys := []string{}
	gcpLoader := cfgcp.GCPLoader{}
	//we can possibly also just list project directly from the api here
	gcpProjects, err := gcpLoader.Load()
	if err != nil {
		return err
	}

	projectKeys = append(projectKeys, gcpProjects...)
	if configName == "" {
		withStdio := survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)

		clio.NewLine()
		// Replicate the logic from original assume fn.
		if len(projectKeys) == 0 {
			return clierr.New("Granted couldn't find any GCP profiles in your config file or your credentials file")
		}
		in := survey.Select{
			Message: "Please select the project you would like to assume:",
			Options: projectKeys,
			Filter:  filterMultiToken,
		}

		err = testable.AskOne(&in, &configName, withStdio)
		if err != nil {
			return err
		}
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	//look up the project name from the config as the name isnt always the project name
	config, err := gcpLoader.Get(configName)
	if err != nil {
		return err
	}

	filePath := ""

	if a.assumeFlags.Bool("impersonate") {
		serviceAccountKeys := []string{}

		serviceAccounts, err := cfgcp.LoadServiceAccounts(ctx.Context, config.Project)
		if err != nil {
			return err
		}

		var serviceAccount string

		for _, serviceAccount := range serviceAccounts {
			serviceAccountKeys = append(serviceAccountKeys, serviceAccount.Email)
		}

		withStdio := survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)

		clio.NewLine()

		if len(serviceAccountKeys) == 0 {
			return clierr.New("Granted couldn't find any GCP Service Accounts for this project. You probably aren't assigned to any.")
		}
		in := survey.Select{
			Message: "Please select the service account you would like to impersonate:",
			Options: serviceAccountKeys,
			Filter:  filterMultiToken,
		}

		err = testable.AskOne(&in, &serviceAccount, withStdio)
		if err != nil {
			return err
		}

		sa := cfgcp.ServiceAccount{
			Name:      serviceAccount,
			Type:      "GCP_SERVICE_ACCOUNT_IMPERSONATION",
			ProjectId: config.Project,
		}

		credsJson, err := sa.AssumeTerminal(ctx.Context)
		if err != nil {
			return err
		}

		//save credentials into ~/.config/gcloud/application_default_credentials.json
		//It doesn't seem like we are able to save these credentials to the application default
		//for now we will just return the short lived access token to the user if they request it

		clio.Info(string(credsJson))
		// homeDir, err := os.UserHomeDir()
		// if err != nil {
		// 	return err
		// }
		// // Define the file path
		// filePath = homeDir + "/.config/gcloud/application_default_credentials.json"

		// // Write the JSON data to the file
		// err = ioutil.WriteFile(filePath, credsJson, 0644)
		// if err != nil {
		// 	return err
		// }
	}

	if os.Getenv("GRANTED_QUIET") != "true" {
		clio.Successf("Set GCP config and project %s %s", configName, config.Project)
	}

	output := PrepareStringsForShellScript([]string{configName, config.Project, config.Account, config.Region, config.Zone, filePath})
	fmt.Printf("GrantedImpersonateSA %s %s %s %s %s %s", output...)

	if a.getConsoleURL {

		consoleURL := fmt.Sprintf("https://console.cloud.google.com?project=%s&authuser=%s", config.Project, config.Account)

		containerProfile := configName

		if a.assumeFlags.String("browser-profile") != "" {
			containerProfile = a.assumeFlags.String("browser-profile")
		}

		browserPath := cfg.CustomBrowserPath
		if browserPath == "" {
			return errors.New("default browser not configured. run `granted browser set` to configure")
		}

		l := launcher.Open{}

		clio.Infof("Opening a console for %s in your browser...", config.Project)

		// now build the actual command to run - e.g. 'firefox --new-tab <URL>'
		args := l.LaunchCommand(consoleURL, containerProfile)

		var startErr error
		if l.UseForkProcess() {
			clio.Debugf("running command using forkprocess: %s", args)
			cmd, err := forkprocess.New(args...)
			if err != nil {
				return err
			}
			startErr = cmd.Start()
		} else {
			clio.Debugf("running command without forkprocess: %s", args)
			cmd := exec.Command(args[0], args[1:]...)
			startErr = cmd.Start()
		}

		if startErr != nil {
			return clierr.New(fmt.Sprintf("Granted was unable to open a browser session automatically due to the following error: %s", startErr.Error()),
				// allow them to try open the url manually
				clierr.Info("You can open the browser session manually using the following url:"),
				clierr.Info(consoleURL),
			)
		}
	}
	return nil

}

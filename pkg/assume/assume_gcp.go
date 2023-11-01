package assume

import (
	"fmt"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/common-fate/clio"
	"github.com/common-fate/clio/clierr"
	"github.com/common-fate/granted/pkg/cfgcp"
	"github.com/common-fate/granted/pkg/testable"
	cfflags "github.com/common-fate/granted/pkg/urfav_overrides"
	"github.com/urfave/cli/v2"
)

type AssumeGCP struct {
	ctx           *cli.Context
	assumeFlags   *cfflags.Flags
	getConsoleURL bool
}

// processArgsAndExecFlag will return the profileName if provided and the exec command config if the exec flag is used
// this supports both the -- variant and the legacy flag when passes the command and args as a string for backwards compatability
func (a AssumeGCP) processArgsAndExecFlag(c *cli.Context, assumeFlags *cfflags.Flags) (string, *execConfig, error) {
	execFlag := assumeFlags.String("exec")
	clio.Debugw("process args", "execFlag", execFlag, "osargs", os.Args, "c.args", c.Args().Slice())
	if execFlag == "" {
		if len(c.Args().Slice()) == 1 {
			return "", nil, nil
		}
		return c.Args().Slice()[1], nil, nil
	}

	if execFlag == "--" {
		for i, arg := range os.Args {
			if arg == "--" {
				if len(os.Args) == i+1 {
					return "", nil, clierr.New("invalid arguments to exec call with '--'. Make sure you pass the command and argument after the doubledash.",
						clierr.Info("try running 'assume profilename --exec -- cmd arg1 arg2"))
				}
				cmdAndArgs := os.Args[i+1:]
				var args []string
				if len(cmdAndArgs) > 1 {
					args = cmdAndArgs[1:]
				}
				if c.Args().Len() > len(cmdAndArgs) {
					return c.Args().First(), &execConfig{cmdAndArgs[0], args}, nil
				} else {
					return "", &execConfig{cmdAndArgs[0], args}, nil
				}
			}
		}
	}

	parts := strings.SplitN(execFlag, " ", 2)
	var args []string
	if len(parts) > 1 {
		args = strings.Split(parts[1], " ")
	}
	return c.Args().Slice()[1], &execConfig{parts[0], args}, nil
}

func (a AssumeGCP) Assume() error {

	configName, _, err := a.processArgsAndExecFlag(a.ctx, a.assumeFlags)
	if err != nil {
		return err
	}

	projectKeys := []string{}
	gcpLoader := cfgcp.GCPLoader{}
	gcpProjects, err := gcpLoader.Load()
	if err != nil {
		return err
	}

	projectKeys = append(projectKeys, gcpProjects...)
	if configName == "" {
		withStdio := survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)

		clio.NewLine()
		// Replicate the logic from original assume fn.
		in := survey.Select{
			Message: "Please select the project you would like to assume:",
			Options: projectKeys,
			Filter:  filterMultiToken,
		}
		if len(projectKeys) == 0 {
			return clierr.New("Granted couldn't find any AWS profiles in your config file or your credentials file",
				clierr.Info("You can add profiles to your AWS config by following our guide: "),
				clierr.Info("https://docs.commonfate.io/granted/getting-started#set-up-your-aws-profile-file"),
			)
		}

		err = testable.AskOne(&in, &configName, withStdio)
		if err != nil {
			return err
		}
	}

	if a.getConsoleURL {
		//example url
		//https://console.cloud.google.com/start?authuser=1&project=cf-dev-368022
		//https://console.cloud.google.com/welcome?project=develop-403601&serviceId=default
		//https://console.cloud.google.com/welcome?serviceId=default&authuser=1
		//https://console.cloud.google.com/welcome?project=develop-403601&authuser=1
	}

	//look up the project name from the config as the name isnt always the project name
	config, err := gcpLoader.Get(configName)
	if err != nil {
		return err
	}

	// fmt.Printf("%v", config)
	//set the project environment variable
	fmt.Printf("GrantedGCPProject %s %s %s %s %s", configName, config.Project, config.Account, config.Region, config.Zone)

	clio.Success("Updated config and current project")
	clio.Info("Config: ", configName)
	clio.Info("Project: ", config.Project)
	clio.Info("Account: ", config.Account)
	clio.Info("Region: ", config.Region)
	clio.Info("Zone: ", config.Zone)

	return nil
}

package granted

import (
	"errors"
	"fmt"
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/common-fate/granted/pkg/browser"
	"github.com/common-fate/granted/pkg/config"
	"github.com/common-fate/granted/pkg/forkprocess"
	"github.com/common-fate/granted/pkg/launcher"
	"github.com/urfave/cli/v2"
)

var WorkspaceCommand = cli.Command{
	Name:        "workspace",
	Usage:       "Launch a browser workspace",
	Subcommands: []*cli.Command{&WorkspaceAddCommand},
	Flags: []cli.Flag{
		&cli.BoolFlag{Name: "temporary", Aliases: []string{"t"}},
		&cli.IntFlag{Name: "sessions", Aliases: []string{"s"}},
	},
	Action: func(c *cli.Context) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		browserPath := cfg.CustomBrowserPath
		if browserPath == "" {
			return errors.New("default browser not configured. run `granted browser set` to configure")
		}
		type Launcher interface {
			LaunchCommand(url string, profile string) []string
		}
		grantedFolder, err := config.GrantedConfigFolder()
		if err != nil {
			return err
		}
		var l Launcher
		switch cfg.DefaultBrowser {
		case browser.ChromeKey:
			l = launcher.ChromeProfile{
				ExecutablePath: browserPath,
				UserDataPath:   path.Join(grantedFolder, "chromium-profiles", "1"), // held over for backwards compatibility, "1" indicates Chrome profiles
			}
		case browser.BraveKey:
			l = launcher.ChromeProfile{
				ExecutablePath: browserPath,
				UserDataPath:   path.Join(grantedFolder, "chromium-profiles", "2"), // held over for backwards compatibility, "2" indicates Brave profiles
			}
		case browser.EdgeKey:
			l = launcher.ChromeProfile{
				ExecutablePath: browserPath,
				UserDataPath:   path.Join(grantedFolder, "chromium-profiles", "3"), // held over for backwards compatibility, "3" indicates Edge profiles
			}
		case browser.ChromiumKey:
			l = launcher.ChromeProfile{
				ExecutablePath: browserPath,
				UserDataPath:   path.Join(grantedFolder, "chromium-profiles", "4"), // held over for backwards compatibility, "4" indicates Chromium profiles
			}
		case browser.FirefoxKey:
			l = launcher.Firefox{
				ExecutablePath: browserPath,
			}
		default:
			l = launcher.Open{}
		}

		if c.Bool("temporary") {
			for i := 0; i < c.Int("sessions"); i++ {
				profile := strings.Join([]string{"granted-workspace-temporary", strconv.Itoa(i)}, "-")
				url := fmt.Sprintf("ext+granted-containers:name=%s&url=%s", profile, url.QueryEscape("http://localhost:3000"))
				args := l.LaunchCommand(url, profile)
				cmd, err := forkprocess.New(args...)
				if err != nil {
					return err
				}
				err = cmd.Start()
				if err != nil {
					return err
				}
			}
			return nil
		}

		workspaceArgument := c.Args().First()
		if cfg.Workspaces == nil {
			return fmt.Errorf("no workspaces configured yet, configure one now! ")
		}

		workspaces := []string{}
		for _, w := range cfg.Workspaces.Workspaces {
			workspaces = append(workspaces, w.ID)
		}
		if workspaceArgument == "" {
			in := survey.Select{
				Message: "Please select the workspace you would like to launch",
				Options: workspaces,
			}

			err = survey.AskOne(&in, &workspaceArgument)
			if err != nil {
				return err
			}
		}
		for _, workspace := range cfg.Workspaces.Workspaces {
			if workspace.ID == workspaceArgument {
				for _, s := range workspace.Sessions {
					profile := strings.Join([]string{"granted-workspace", workspaceArgument, s.Name}, "-")
					url := fmt.Sprintf("ext+granted-containers:name=%s&url=%s", profile, url.QueryEscape(workspace.URL))
					args := l.LaunchCommand(url, profile)
					cmd, err := forkprocess.New(args...)
					if err != nil {
						return err
					}
					err = cmd.Start()
					if err != nil {
						return err
					}
				}
				return nil
			}
		}
		return fmt.Errorf("no matching workspace found")
	},
}

var WorkspaceAddCommand = cli.Command{
	Name:  "add",
	Usage: "Add a browser workspace",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "id"},
		&cli.StringFlag{Name: "url"},
		&cli.StringSliceFlag{Name: "session"},
	},
	Action: func(c *cli.Context) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		if cfg.Workspaces == nil {
			cfg.Workspaces = &config.WorkspaceConfig{}
		}

		id := c.String("id")
		url := c.String("url")
		sessions := c.StringSlice("session")
		if id == "" {
			in := survey.Input{
				Message: "Please enter an ID for this workspace",
			}
			err = survey.AskOne(&in, &id, survey.WithValidator(func(ans interface{}) error {
				a := ans.(string)
				for _, workspace := range cfg.Workspaces.Workspaces {
					if workspace.ID == a {
						return fmt.Errorf("workspace already exists")
					}
				}
				return nil
			}))
			if err != nil {
				return err
			}
		}
		if url == "" {
			in := survey.Input{
				Message: "Please enter a URL for this workspace",
			}
			err = survey.AskOne(&in, &url)
			if err != nil {
				return err
			}
		}
		workspace := config.Workspace{
			ID:  id,
			URL: url,
		}
		if len(sessions) == 0 {
			session := ""
			for session != "end" {
				in := survey.Input{
					Message: "Please enter a name for the session, you may create as many sessions as you need. Type 'end' to finish creating sessions.",
				}
				err = survey.AskOne(&in, &session, survey.WithValidator(func(ans interface{}) error {
					a := ans.(string)
					for _, session := range sessions {
						if session == a {
							return fmt.Errorf("session already exists")
						}
					}
					return nil
				}))
				if err != nil {
					return err
				}
				if session != "end" {
					sessions = append(sessions, session)
				}
			}
		}

		for _, s := range sessions {
			workspace.Sessions = append(workspace.Sessions, config.WorkspaceSession{Name: s})
		}
		cfg.Workspaces.Workspaces = append(cfg.Workspaces.Workspaces, workspace)

		return cfg.Save()
	},
}

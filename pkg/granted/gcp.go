package granted

import (
	"database/sql"
	"fmt"
	"html/template"
	"os"
	"path"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/common-fate/granted/pkg/testable"
	"github.com/urfave/cli/v2"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/option"
	"gopkg.in/ini.v1"

	_ "github.com/mattn/go-sqlite3"
)

const CONFIG_TEMPLATE = `
[core]
project = {{ .Project }}
account = {{ .Account }}
`

type CoreConfig struct {
	Project string
	Account string
}

var GCPCommand = cli.Command{
	Name:        "gcp",
	Subcommands: []*cli.Command{&GenerateSubcommand, &ListConfigSubcommand},
}

var ListConfigSubcommand = cli.Command{
	Name:  "list",
	Usage: "will list all available gcloud configs",
	Action: func(ctx *cli.Context) error {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}

		dirpath := path.Join(home, ".config", "gcloud", "configurations")

		configs, err := os.ReadDir(dirpath)
		if err != nil {
			return err
		}

		gcloudCfgs := []string{}

		for _, cfg := range configs {
			if strings.HasPrefix(cfg.Name(), "config_") {
				gcloudCfgs = append(gcloudCfgs, strings.TrimPrefix(cfg.Name(), "config_"))
			}
		}

		var gConfigName string
		in := survey.Select{
			Message: "Please select the profile you would like to assume:",
			Options: gcloudCfgs,
		}

		err = testable.AskOne(&in, &gConfigName)
		if err != nil {
			return err
		}

		for _, cfg := range configs {
			if cfg.Name() == "config_"+gConfigName {

				// read the file
				f, err := os.ReadFile(path.Join(dirpath, cfg.Name()))
				if err != nil {
					return err
				}

				cfg, err := ini.Load([]byte(string(f)))
				if err != nil {
					return err
				}

				core, err := cfg.GetSection("core")
				if err != nil {
					return err
				}

				projectId, err := core.GetKey("project")
				if err != nil {
					return err
				}

				accountId, err := core.GetKey("account")
				if err != nil {
					return err
				}

				// need to export them as environment variable
				fmt.Printf("the selected project-id='%s' account-id='%s'", projectId, accountId)
			}
		}

		return nil
	},
}

var GenerateSubcommand = cli.Command{
	Name:  "generate",
	Usage: "Will generate configuration for all the projects user have access to",
	Action: func(c *cli.Context) error {
		ctx := c.Context

		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}

		filepath := path.Join(home, ".config", "gcloud", "credentials.db")
		db, err := sql.Open("sqlite3", filepath)
		if err != nil {
			return err
		}

		rows, err := db.Query("SELECT account_id FROM credentials")
		if err != nil {
			return err
		}
		defer rows.Close()

		accountIds := []string{}
		for rows.Next() {
			var accountId string
			err := rows.Scan(&accountId)
			if err != nil {
				return err
			}

			accountIds = append(accountIds, accountId)
		}

		for _, accountId := range accountIds {
			creds_filepath := path.Join(home, ".config", "gcloud", "legacy_credentials", accountId, "adc.json")
			client, err := cloudresourcemanager.NewService(ctx, option.WithCredentialsFile(creds_filepath))
			if err != nil {
				return err
			}

			projects, err := client.Projects.List().Context(ctx).Do()
			if err != nil {
				return err
			}

			for _, project := range projects.Projects {

				t, err := template.New("").Parse(CONFIG_TEMPLATE)
				if err != nil {
					return err
				}

				configName := project.ProjectId
				filepath := path.Join(home, ".config", "gcloud", "configurations")

				file, err := os.OpenFile(path.Join(filepath, "config_"+configName), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
				if err != nil {
					return err
				}

				err = t.Execute(file, CoreConfig{
					Project: project.ProjectId,
					Account: accountId,
				})
				if err != nil {
					return err
				}
			}
		}

		return nil
	},
}

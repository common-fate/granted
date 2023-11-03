package granted

import (
	"database/sql"
	"fmt"
	"html/template"
	"os"
	"path"

	"github.com/common-fate/clio"
	"github.com/common-fate/granted/pkg/cfgcp"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/option"

	_ "github.com/mattn/go-sqlite3"
)

const CONFIG_TEMPLATE = `[core]
project = {{ .Project }}
account = {{ .Account }}
{{- if or .Region .Zone}}
[zone]
{{- if .Region }}
region = {{ .Region }}
{{- end}}
{{- if .Zone }}
zone = {{ .Zone }}
{{- end}}
{{- end}}`

type CoreConfig struct {
	Project string
	Account string
	Region  string
	Zone    string
}

var GCPCommand = cli.Command{
	Name:        "gcp",
	Usage:       "Google Cloud Platform (GCP) specific actions",
	Subcommands: []*cli.Command{&GenerateSubcommand},
}

var GenerateSubcommand = cli.Command{
	Name:  "generate",
	Usage: "Generate configuration for all the projects user have access to in GCP based on active credentials available.",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "region", Usage: "Provide region to populate zone/region in the config"},
		&cli.StringFlag{Name: "zone", Usage: "Provide zone to populate zone/zone in the config"},
	},
	Action: func(c *cli.Context) error {
		ctx := c.Context

		gcp := cfgcp.GCPLoader{}

		configPath, err := gcp.GetOSSpecifcConfigPath()
		if err != nil {
			return err
		}

		filepath := path.Join(configPath, "credentials.db")
		db, err := sql.Open("sqlite3", filepath)
		if err != nil {
			return err
		}

		rows, err := db.Query("SELECT account_id FROM credentials")
		if err != nil {
			return errors.Wrap(err, "querying credentials.db get all account_id")
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

		if len(accountIds) == 0 {
			clio.Warn("We could not find any authenticated accounts.")
			clio.Info("Try running 'gcloud auth login' before using this command.")
		}

		projectCount := 0
		for _, accountId := range accountIds {
			creds_filepath := path.Join(configPath, "legacy_credentials", accountId, "adc.json")
			client, err := cloudresourcemanager.NewService(ctx, option.WithCredentialsFile(creds_filepath))
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("creating new cloudresourcemanager service with legacy_credentials file for accountId=%s", accountId))
			}

			projects, err := client.Projects.List().Context(ctx).Do()
			if err != nil {
				return err
			}

			for _, project := range projects.Projects {
				projectCount++

				t, err := template.New("").Parse(CONFIG_TEMPLATE)
				if err != nil {
					return errors.Wrap(err, "parsing template file")
				}

				configName := project.ProjectId
				filepath := path.Join(configPath, "configurations")

				file, err := os.OpenFile(path.Join(filepath, "config_"+configName), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
				if err != nil {
					return err
				}

				err = t.Execute(file, CoreConfig{
					Project: project.ProjectId,
					Account: accountId,
					Region:  c.String("region"),
					Zone:    c.String("zone"),
				})
				if err != nil {
					return err
				}
			}
		}

		clio.Successf("Successfully generated gcloud configurations for %d projects and %d accounts", projectCount, len(accountIds))
		clio.Infof("Run 'gcloud config configurations list' to see all the generated profiles")
		clio.Infof("Run 'assume gcp <tab>' to assume into one of the config")

		return nil
	},
}

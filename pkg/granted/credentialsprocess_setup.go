package granted

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/bigkevmcd/go-configparser"
	"github.com/common-fate/granted/pkg/cfaws"
	grantedConfig "github.com/common-fate/granted/pkg/config"
	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
)

var ConfigSetup = cli.Command{
	Name:  "setup",
	Usage: "Alters your AWS Config credential_processs",
	Action: func(c *cli.Context) error {

		withStdio := survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)
		in := &survey.Input{
			Message: "What is the base url of your Granted Approvals deployment\n",
			Help:    "Enter a base URL without trailing slash i.e. https://commonfate.approvals.dev",
		}
		var url string
		err := survey.AskOne(in, &url, withStdio)
		if err != nil {
			return err
		}
		if url == "" {
			return errors.New("cancelled setup process")
		}
		gConf, err := grantedConfig.Load()
		if err != nil {
			return errors.New("failed to load Config for GrantedApprovalsUrl")
		}
		gConf.GrantedApprovalsUrl = url
		err = gConf.Save()
		if err != nil {
			return err
		}

		p := cfaws.Profiles{Profiles: make(map[string]*cfaws.Profile)}
		// Load the default config
		configPath := config.DefaultSharedConfigFilename()
		configFile, err := configparser.NewConfigParserFromFile(configPath)

		if err != nil {
			fmt.Println("Error loading config file")
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}

		// Also add a warning for large config files
		if len(configFile.Sections()) > 100 {
			fmt.Println("Warning: your AWS config file has >100 sections")
			fmt.Println("Ordering can change, comments may be removed. Please backup your file if this is a concern.")
		}

		fmt.Println("This will add new sections to your aws config")
		withStdio = survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)
		confirmIn := &survey.Confirm{
			Message: "Would you like to proceed?",
			Default: true,
		}
		var confirm bool
		err = survey.AskOne(confirmIn, &confirm, withStdio)
		if err != nil {
			return err
		}
		if !confirm {
			return errors.New("cancelled setup process")
		}

		newConfigSections := configparser.New()

		// Itterate over each section
		for _, section := range configFile.Sections() {
			rawConfig, err := configFile.Items(section)

			if err != nil {
				fmt.Fprintf(color.Error, "failed to parse a profile from your AWS config: %s Due to the following error: %s\n", section, err)
				continue
			}
			// Check if the section is prefixed with 'profile ' and that the profile has a name
			if ((strings.HasPrefix(section, "profile ") && len(section) > 8) || section == "default") && cfaws.IsLegalProfileName(section) {
				name := strings.TrimPrefix(section, "profile ")
				p.ProfileNames = append(p.ProfileNames, name)
				p.Profiles[name] = &cfaws.Profile{RawConfig: rawConfig, Name: name, File: configPath}

				// Now append this to the new config
				s := "profile granted." + name
				newConfigSections.AddSection(s)
				newConfigSections.Set(s, "credential_process", "dgranted credentialsprocess --profile "+name)
				// Write to credential_process our custom script
				configFile.AddSection(s)
				configFile.Set(s, "credential_process", "dgranted credentialsprocess --profile "+name)
			}
		}
		// This is just a secondary backup to +2
		configFile.SaveWithDelimiter(configPath, "=")
		newConfigSections.SaveWithDelimiter(configPath+"2", "=")
		if err != nil {
			fmt.Println(err.Error())
		}

		// Done :)

		return nil
	},
}
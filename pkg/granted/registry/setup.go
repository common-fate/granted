package registry

import (
	"fmt"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/common-fate/clio"
	"github.com/common-fate/clio/clierr"

	"github.com/urfave/cli/v2"
)

var SetupCommand = cli.Command{
	Name:        "setup",
	Description: "Setup a granted registry for the first time",
	Subcommands: []*cli.Command{},
	Action: func(c *cli.Context) error {
		// check that it is an empty dir
		err := ensureConfigDoesntExist(c)
		if err != nil {
			return err
		}

		// copy ~/.aws/config to ./config
		configFile, err := loadAWSConfigFile()
		if err != nil {
			return err
		}

		var confirm bool
		s := &survey.Confirm{
			Message: "Are you sure you want to save your credentials file to the current directory?",
			Default: true,
		}
		err = survey.AskOne(s, &confirm)
		if err != nil {
			return err
		}
		if !confirm {
			fmt.Println("Cancelled registry setup")
			return nil
		}

		// now save cfg contents to ./config
		err = configFile.SaveTo("./config")
		if err != nil {
			return err
		}

		// create granted.yml
		f, err := os.Create("granted.yml")
		if err != nil {
			return err
		}

		// now initialize the git repo
		err = gitInit("./")
		if err != nil {
			return err
		}
		// write the default config to the granted.yml
		_, err = f.WriteString(`awsConfig:
		- ./config`)
		if err != nil {
			return err
		}
		err = f.Close()
		if err != nil {
			return err
		}

		clio.Info(`Registry setup complete.`)

		return nil
	},
}

// sanity check: verify that a config file doesn't already exist.
// if it does, the user may have run this command by mistake.
func ensureConfigDoesntExist(c *cli.Context) error {
	_, err := os.Open("granted.yml")
	if err != nil {
		return nil
	}

	// overwrite := c.Bool("overwrite")
	// if overwrite {
	// 	clio.Warnf("--overwrite has been set, the config file %s will be overwritten", f)
	// 	return nil
	// }

	// if we get here, the config file exists and is at risk of being overwritten.
	return clierr.New(("A granted.yml file already exists in this folder.\ngranted will exit to avoid overwriting this file, in case you've run this command by mistake."),
		clierr.Info(`Alternatively, take one of the following actions:
  a) run 'granted registry setup' from a different folder
  b) run 'granted registry sync' to instead make updates to the existing registry
`))
}

var PushCommand = cli.Command{
	Name:        "push",
	Description: "Push changes to the registry",
	Action: func(c *cli.Context) error {

		// Check if a remote has been added to the repo
		hasRemote, err := gitHasRemote("./")
		if err != nil {
			return err
		}
		fmt.Println("hasRemote", hasRemote)

		return nil
	},
}

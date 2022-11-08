package registry

import (
	"fmt"
	"os"

	"github.com/common-fate/clio/clierr"

	"github.com/urfave/cli/v2"
)

var SetupCommand = cli.Command{
	Name:        "setup",
	Description: "Setup a granted registry for the first time",
	// Flags:       []cli.Flag{&cli.PathFlag{Name: "path", Aliases: []string{"p"}, Usage: "Path to the registry directory", Required: true}},
	Subcommands: []*cli.Command{&PushCommand},
	Action: func(c *cli.Context) error {

		// check that it is an empty dir
		err := ensureConfigDoesntExist(c)
		if err != nil {
			return err
		}

		// path := c.Path("path")

		// create granted.yml
		f, err := os.Create("granted.yml")
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

		// copy ~/.aws/config to ./config
		configFile, err := loadAWSConfigFile()
		if err != nil {
			return err
		}
		// now save cfg contents to ./config
		err = configFile.SaveTo("./config")
		if err != nil {
			return err
		}

		// now initialize the git repo
		err = gitInit("./")
		if err != nil {
			return err
		}

		// if c.Args().Len() < 1 {
		// 	clio.Error("Please provide a git repository you want to add like 'granted registry add <https://github.com/your-org/your-registry.git>'")
		// }

		// var repoURLs []string

		// n := 0
		// for n < c.Args().Len() {
		// 	repoURLs = append(repoURLs, c.Args().Get(n))
		// 	n++
		// }

		// Final steps involved
		// Create a repo on github,
		// Add the remote to the local repo
		// git push origin master
		// 		(also need to check that remote has been set)

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
		clierr.Info(`
To fix this, take one of the following actions:
  a) run 'granted registry setup' from a different folder
  b) run 'granted registry sync' to instead make updates to the existing registry
`))
}

var PushCommand = cli.Command{
	Name:        "push",
	Description: "Push changes to the registry",
	Action: func(c *cli.Context) error {

		// Check if a remote has been added to the repo
		// if not, add it

		hasRemote, err := gitHasRemote("./")
		if err != nil {
			return err
		}
		fmt.Println("hasRemote", hasRemote)

		// git add .
		// git commit -m "init"
		// git push origin master

		return nil
	},
}

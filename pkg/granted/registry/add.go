package registry

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"

	grantedConfig "github.com/common-fate/granted/pkg/config"
	"github.com/urfave/cli/v2"
)

var AddCommand = cli.Command{
	Name: "add",
	Action: func(c *cli.Context) error {

		if c.Args().Len() != 1 {
			return fmt.Errorf("git repository not provided. You need to provide a git repository like 'granted add https://github.com/your-org/your-registry.git'")
		}

		repoURL := c.Args().First()
		fmt.Printf("git clone %s\n", repoURL)

		u, err := url.ParseRequestURI(repoURL)
		if err != nil {
			return errors.New(err.Error())
		}

		repoDirPath, err := GetRegistryLocation(u)
		if err != nil {
			return err
		}

		cmd := exec.Command("git", "clone", repoURL, repoDirPath)

		err = cmd.Run()
		if err != nil {
			// TODO: Will throw an error if the folder already exists and is not an empty directory.
			fmt.Println("the error is", err)
			return err
		}

		fmt.Println("Sucessfully cloned the repo")
		gConf, err := grantedConfig.Load()
		if err != nil {
			return err
		}

		// save the repo url to granted config toml file.
		gConf.ProfileRegistryURL = repoURL

		if err := gConf.Save(); err != nil {
			return err
		}

		if err, ok := isValidRegistry(repoDirPath, repoURL); err != nil || !ok {
			if err != nil {
				return err
			}

			return fmt.Errorf("unable to find `granted.yml` file in %s", repoURL)
		}

		var r Registry
		_, err = r.Parse(repoDirPath)
		if err != nil {
			return err
		}

		// TODO: Run Sync logic here.

		return nil
	},
}

func formatFolderPath(p string) string {
	var formattedURL string = ""

	// remove trailing whitespaces.
	formattedURL = strings.TrimSpace(p)

	// remove trailing '/'
	formattedURL = strings.TrimPrefix(formattedURL, "/")

	// remove .git from the folder name
	formattedURL = strings.Replace(formattedURL, ".git", "", 1)

	return formattedURL
}

func isValidRegistry(folderpath string, url string) (error, bool) {
	dir, err := os.ReadDir(folderpath)
	if err != nil {
		return err, false
	}

	for _, file := range dir {
		if file.Name() == "granted.yml" || file.Name() == "granted.yaml" {
			return nil, true
		}
	}

	return nil, false
}

package registry

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"

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

		//grab out the subpath if there is one
		//Will have the format like this https://github.com/octo-org/granted-registry.git/team_a/granted.yml
		var subpath string
		split := strings.Split(repoURL, ".git")
		if len(split) > 1 {
			repoURL = split[0] + ".git"
			subpath = split[1]
		} else {
			repoURL = split[0] + ".git"
		}
		//TODO: subpath will then be used when syncing to only sync from the specified subpath of the repo into the aws config
		_ = subpath

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

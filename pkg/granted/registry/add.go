package registry

import (
	"fmt"
	"os"
	"strings"

	"github.com/common-fate/clio"
	grantedConfig "github.com/common-fate/granted/pkg/config"

	"github.com/urfave/cli/v2"
)

// Prevent issues where these flags are initialised in some part of the program then used by another part
// For our use case, we need fresh copies of these flags in the app and in the assume command
// we use this to allow flags to be set on either side of the profile arg e.g `assume -c profile-name -r ap-southeast-2`
func GlobalFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{Name: "ref", Aliases: []string{"r"}, Usage: "Used to reference a specific commit hash, tag name or branch name"},
	}
}

var AddCommand = cli.Command{
	Name:        "add",
	Description: "Add a profile registry that you want to sync with aws config file",
	Usage:       "Provide git repository you want to sync with aws config file",
	Flags:       GlobalFlags(),
	Action: func(c *cli.Context) error {

		if c.Args().Len() < 1 {
			clio.Error("Please provide a git repository you want to add like 'granted registry add <https://github.com/your-org/your-registry.git>'")
		}

		var repoURLs []string

		n := 0
		for n < c.Args().Len() {
			repoURLs = append(repoURLs, c.Args().Get(n))
			n++
		}

		// TODO: grab out the subpath if there is one
		// Will have the format like this https://github.com/octo-org/granted-registry.git/team_a/granted.yml
		// var subpath string
		// split := strings.Split(repoURL, ".git")
		// if len(split) > 1 {
		// 	repoURL = split[0] + ".git"
		// 	subpath = split[1]
		// } else {
		// 	repoURL = split[0] + ".git"
		// }
		// //TODO: subpath will then be used when syncing to only sync from the specified subpath of the repo into the aws config
		// _ = subpath

		gConf, err := grantedConfig.Load()
		if err != nil {
			return err
		}

		for index, repoURL := range repoURLs {
			url, err := parseGitURL(repoURL)
			if err != nil {
				return err
			}

			repoDirPath, err := getRegistryLocation(url)
			if err != nil {
				return err
			}

			if _, err = os.Stat(repoDirPath); err != nil {
				// directory doesn't exist; clone the repo
				if os.IsNotExist(err) {
					err = gitClone(repoURL, repoDirPath)
					if err != nil {
						return err
					}

					// //if a specific ref is passed we will checkout that ref
					// if addFlags.String("ref") != "" {
					// 	fmt.Println("attempting to checkout branch" + addFlags.String("ref"))

					// 	err = checkoutRef(addFlags.String("ref"), repoDirPath)
					// 	if err != nil {
					// 		return err

					// 	}
					// }

				} else {
					// other error. Should not happen.
					return err
				}
			} else {
				// file exists; pull instead of clone.
				clio.Debugf("%s already exists; pulling instead of cloning. ", repoURL)
				gitPull(repoDirPath, false)
			}

			//if a specific ref is passed we will checkout that ref
			// if addFlags.String("ref") != "" {
			// 	fmt.Println("attempting to checkout branch" + addFlags.String("ref"))
			// 	err = checkoutRef(addFlags.String("ref"), repoDirPath)
			// 	if err != nil {
			// 		return err

			// 	}
			// }

			// check if the fetched cloned repo contains granted.yml file.
			if err = parseClonedRepo(repoDirPath, repoURL); err != nil {
				return err
			}

			var r Registry
			_, err = r.Parse(repoDirPath)
			if err != nil {
				return err
			}

			// save only if new repo url is added.
			// TODO: ssh & https for the same repo will duplicate.
			if !Contains(gConf.ProfileRegistryURLS, repoURL) {
				gConf.ProfileRegistryURLS = append(gConf.ProfileRegistryURLS, repoURL)

				if err := gConf.Save(); err != nil {
					return err
				}
			}

			isFirstSection := false
			if index == 0 {
				isFirstSection = true
			}

			// Sync clonned repo content with aws config file.
			if err := Sync(r, repoURL, repoDirPath, isFirstSection); err != nil {
				return err
			}
		}

		return nil
	},
}

func FormatFolderPath(p string) string {
	var formattedURL string

	// remove trailing whitespaces.
	formattedURL = strings.TrimSpace(p)

	// remove trailing '/'
	formattedURL = strings.TrimPrefix(formattedURL, "/")

	// remove .git from the folder name
	formattedURL = strings.Replace(formattedURL, ".git", "", 1)

	return formattedURL
}

func parseClonedRepo(folderpath string, url string) error {
	dir, err := os.ReadDir(folderpath)
	if err != nil {
		return err
	}

	for _, file := range dir {
		if file.Name() == "granted.yml" || file.Name() == "granted.yaml" {
			return nil
		}
	}

	return fmt.Errorf("unable to find `granted.yml` file in %s", url)
}

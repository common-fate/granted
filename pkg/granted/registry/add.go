package registry

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"

	grantedConfig "github.com/common-fate/granted/pkg/config"
	cfflags "github.com/common-fate/granted/pkg/urfav_overrides"

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
	Name:  "add",
	Flags: GlobalFlags(),
	Action: func(c *cli.Context) error {

		addFlags, err := cfflags.New("assumeFlags", GlobalFlags(), c, 3)
		if err != nil {
			return err
		}

		if c.Args().Len() < 1 {
			return fmt.Errorf("git repository not provided. You need to provide a git repository like 'granted add https://github.com/your-org/your-registry.git'")
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

		// save the repo url to granted config toml file.
		gConf.ProfileRegistryURLS = repoURLs
		if err := gConf.Save(); err != nil {
			return err
		}

		for index, repoURL := range repoURLs {
			// TODO: parse fails for SSH url
			u, err := url.Parse(repoURL)
			if err != nil {
				return errors.New(err.Error())
			}

			repoDirPath, err := GetRegistryLocation(u)
			if err != nil {
				return err
			}

			// check repo directory to see if repo exists
			// use clone if not exists, pull if exists
			_, err = os.Stat(repoDirPath)
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Printf("git clone %s\n", repoURL)

					cmd := exec.Command("git", "clone", repoURL, repoDirPath)

					err = cmd.Run()
					if err != nil {
						return err

					}
					fmt.Println("Successfully cloned the repo")

					//if a specific ref is passed we will checkout that ref
					if addFlags.String("ref") != "" {
						fmt.Println("attempting to checkout branch" + addFlags.String("ref"))

						err = checkoutRef(addFlags.String("ref"), repoDirPath)
						if err != nil {
							return err

						}
					}

				} else {
					return err
				}
			} else {
				//if a specific ref is passed we will checkout that ref
				if addFlags.String("ref") != "" {
					fmt.Println("attempting to checkout branch" + addFlags.String("ref"))
					err = checkoutRef(addFlags.String("ref"), repoDirPath)
					if err != nil {
						return err

					}
				}
				fmt.Printf("git pull %s\n", repoURL)

				cmd := exec.Command("git", "--git-dir", repoDirPath+"/.git", "pull")

				err = cmd.Run()
				if err != nil {
					return err
				}
				fmt.Println("Successfully pulled the repo")

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

func formatFolderPath(p string) string {
	var formattedURL string

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

func checkoutRef(ref string, repoDirPath string) error {
	//if a specific ref is passed we will checkout that ref

	//can be a git hash, tag, or branch name. In that order
	//todo set the path of the repo before checking out

	cmd := exec.Command("git", "checkout", ref)
	cmd.Dir = repoDirPath

	err := cmd.Run()
	if err != nil {
		fmt.Println("the error is", err)
		return err
	}
	fmt.Println("Sucessfully checkout out " + ref)
	return nil

}

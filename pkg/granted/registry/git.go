package registry

import (
	"bufio"
	"fmt"
	"log"
	"os/exec"
	"regexp"

	"github.com/common-fate/clio"
)

type GitURL struct {
	Host string
	Org  string
	Repo string
}

// TODO: Need to test this func
func parseGitURL(repoURL string) (GitURL, error) {
	re := regexp.MustCompile(`((git@|http(s)?:\/\/)(?P<HOST>[\w\.@]+)(\/|:))(?P<ORG>[\w,\-,\_]+)\/(?P<REPO>[\w,\-,\_]+)(.git){0,1}((\/){0,1})`)

	if re.MatchString(repoURL) {
		matches := re.FindStringSubmatch(repoURL)
		hostIndex := re.SubexpIndex("HOST")
		orgIndex := re.SubexpIndex("ORG")
		repoIndex := re.SubexpIndex("REPO")

		return GitURL{
			Host: matches[hostIndex],
			Org:  matches[orgIndex],
			Repo: matches[repoIndex],
		}, nil

	}

	return GitURL{}, fmt.Errorf("unable to parse the provided git url '%s'", repoURL)
}

func gitPull(repoDirPath string, shouldSilentLogs bool) error {
	// pull the repo here.
	clio.Debugf("git -C %s pull %s %s\n", repoDirPath, "origin", "main")
	cmd := exec.Command("git", "-C", repoDirPath, "pull", "origin", "main")

	// StderrPipe returns a pipe that will be connected to the command's
	// standard error when the command starts.
	if shouldSilentLogs {
		stderr, _ := cmd.StderrPipe()
		if err := cmd.Start(); err != nil {
			log.Fatal(err)
		}

		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}
	}

	clio.Infof("Successfully pulled the repo.")

	return nil
}

func gitClone(repoURL string, repoDirPath string) error {
	clio.Debugf("git clone %s\n", repoURL)

	cmd := exec.Command("git", "clone", repoURL, repoDirPath)

	err := cmd.Run()
	if err != nil {
		return err

	}
	clio.Infof("Successfully cloned %s", repoURL)

	return nil
}

// WIP/TODO: set the path of the repo before checking out
// if a specific ref is passed we will checkout that ref
// can be a git hash, tag, or branch name. In that order
func CheckoutRef(ref string, repoDirPath string) error {

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

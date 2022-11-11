package registry

import (
	"bufio"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/common-fate/clio"
)

type GitURL struct {
	ProvidedURL string
	Host        string
	Org         string
	Repo        string
	Subpath     string
	Filename    string
}

// compares if Host, Organization and Repo name is same for both passed URL.
// if there is subfolder and filename then compare if they are same or not.
func IsSameGitURL(a GitURL, b GitURL) bool {
	if a.Filename != "" || b.Filename != "" {
		return (a.Host == b.Host) && (a.Org == b.Org) && (a.Repo == b.Repo) && (a.Subpath == b.Subpath) && (a.Filename == b.Filename)
	}

	if a.Subpath != "" || b.Subpath != "" {
		return (a.Host == b.Host) && (a.Org == b.Org) && (a.Repo == b.Repo) && (a.Subpath == b.Subpath)
	}

	return (a.Host == b.Host) && (a.Org == b.Org) && (a.Repo == b.Repo)
}

func (g *GitURL) GetURL() string {
	split := strings.Split(g.ProvidedURL, ".git")

	return split[0] + ".git"
}

// The passed string is matched with regex that checks whether the string is valid git url
// also checks if the provided string consist of subfolder or yaml spcific path
// the regex checks if optional group (<SUBPATH> or <WITH_YML>) is present.
// if string consist of .y(a)ml then the group will be matched with <WITH_YML> group
// if string doesn't consist of .y(a)ml but does consist of subfolder then <SUBPATH> group will be matched.
func parseGitURL(repoURL string) (GitURL, error) {
	re := regexp.MustCompile(`((git@|http(s)?:\/\/)(?P<HOST>[\w\.@]+)(\/|:))(?P<ORG>[\w,\-,\_]+)\/(?P<REPO>[\w,\-,\_]+)(.git){0,1}(\/)?((?P<WITH_YML>.+ya?ml)|(?P<SUBPATH>(\w+(\/)?)*))`)

	if re.MatchString(repoURL) {
		matches := re.FindStringSubmatch(repoURL)
		hostIndex := re.SubexpIndex("HOST")
		orgIndex := re.SubexpIndex("ORG")
		repoIndex := re.SubexpIndex("REPO")
		subpathIndex := re.SubexpIndex("SUBPATH")
		filepathIndex := re.SubexpIndex("WITH_YML")

		withConfigFilePath := matches[filepathIndex]

		var filename string = ""
		var subpath string = ""

		// filename specific group might not contain subpath
		// example: /team_a/team_b/granted.yml
		// example: .git/granted.yml
		if withConfigFilePath != "" {
			lastIndex := strings.LastIndex(withConfigFilePath, "/")

			// there is a subpath
			if lastIndex > 0 {
				subpath = withConfigFilePath[:lastIndex+1]
			}

			// filename is all the string after the last slash
			filename = withConfigFilePath[lastIndex+1:]
		} else {
			subpath = matches[subpathIndex]
		}

		return GitURL{
			ProvidedURL: repoURL,
			Host:        matches[hostIndex],
			Org:         matches[orgIndex],
			Repo:        matches[repoIndex],
			Subpath:     subpath,
			Filename:    filename,
		}, nil

	}

	return GitURL{}, fmt.Errorf("unable to parse the provided git url '%s'", repoURL)
}

func gitPull(repoDirPath string, shouldSilentLogs bool) error {
	// pull the repo here.
	clio.Debugf("git -C %s pull %s %s\n", repoDirPath, "origin", "HEAD")
	cmd := exec.Command("git", "-C", repoDirPath, "pull", "origin", "HEAD")

	// StderrPipe returns a pipe that will be connected to the command's
	// standard error when the command starts.
	stderr, _ := cmd.StderrPipe()
	if err := cmd.Start(); err != nil {
		return err
	}

	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "error") || strings.Contains(scanner.Text(), "fatal") {
			return fmt.Errorf(scanner.Text())
		}

		if shouldSilentLogs {
			clio.Debug(scanner.Text())
		} else {
			clio.Info(scanner.Text())
		}
	}

	clio.Debugf("Successfully pulled the repo")

	return nil
}

func gitInit(repoDirPath string) error {
	clio.Debugf("git init %s\n", repoDirPath)

	cmd := exec.Command("git", "init", repoDirPath)

	err := cmd.Run()
	if err != nil {
		return err

	}

	return nil
}

func gitClone(repoURL string, repoDirPath string) error {
	clio.Debugf("git clone %s\n", repoURL)

	cmd := exec.Command("git", "clone", repoURL, repoDirPath)

	stderr, _ := cmd.StderrPipe()
	if err := cmd.Start(); err != nil {
		return err
	}

	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "error") || strings.Contains(scanner.Text(), "fatal") {
			return fmt.Errorf(scanner.Text())
		}

		clio.Info(scanner.Text())
	}
	clio.Debugf("Successfully cloned %s", repoURL)

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

func URLExists(arr []string, url GitURL) bool {
	for _, v := range arr {
		u, err := parseGitURL(v)
		// should not happen but if it does let's skip this iteration.
		if err != nil {
			continue
		}

		if IsSameGitURL(u, url) {
			return true
		}

		return false
	}
	return false
}

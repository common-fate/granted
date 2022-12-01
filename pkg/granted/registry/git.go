package registry

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"

	"github.com/common-fate/clio"
)

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
		if strings.Contains(strings.ToLower(scanner.Text()), "error") || strings.Contains(strings.ToLower(scanner.Text()), "fatal") {
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

package git

import (
	"bufio"
	"errors"
	"os/exec"
	"strings"

	"github.com/common-fate/clio"
)

// Pull wraps the command 'git pull'.
func Pull(repoDirPath string, shouldSilentLogs bool) error {
	return PullRef(repoDirPath, "", shouldSilentLogs)
}

// PullRef wraps the command 'git pull' for a specific ref.
func PullRef(repoDirPath string, ref string, shouldSilentLogs bool) error {
	// if ref is specified, ensure we're on the right branch first
	if ref != "" {
		clio.Debugf("git -C %s checkout %s\n", repoDirPath, ref)
		checkoutCmd := exec.Command("git", "-C", repoDirPath, "checkout", ref)
		if err := checkoutCmd.Run(); err != nil {
			return err
		}
	}

	// determine what to pull
	pullRef := "HEAD"
	if ref != "" {
		pullRef = ref
	}

	// pull the repo here.
	clio.Debugf("git -C %s pull %s %s\n", repoDirPath, "origin", pullRef)
	cmd := exec.Command("git", "-C", repoDirPath, "pull", "origin", pullRef)

	// StderrPipe returns a pipe that will be connected to the command's
	// standard error when the command starts.
	stderr, _ := cmd.StderrPipe()
	if err := cmd.Start(); err != nil {
		return err
	}

	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "error") || strings.Contains(scanner.Text(), "fatal") {
			return errors.New(scanner.Text())
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

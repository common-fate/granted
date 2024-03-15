package git

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"

	"github.com/common-fate/clio"
)

// Pull wraps the command 'git pull'.
func Pull(repoDirPath string, shouldSilentLogs bool) error {
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

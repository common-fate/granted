package git

import (
	"bufio"
	"errors"
	"os/exec"
	"strings"

	"github.com/common-fate/clio"
)

// Clone wraps the command 'git clone'.
func Clone(repoURL string, repoDirPath string) error {
	return CloneWithRef(repoURL, repoDirPath, "")
}

// CloneWithRef wraps the command 'git clone' and checks out a specific ref.
func CloneWithRef(repoURL string, repoDirPath string, ref string) error {
	clio.Debugf("git clone %s\n", repoURL)

	cmd := exec.Command("git", "clone", repoURL, repoDirPath)

	stderr, _ := cmd.StderrPipe()
	if err := cmd.Start(); err != nil {
		return err
	}

	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		if strings.Contains(strings.ToLower(scanner.Text()), "error") || strings.Contains(strings.ToLower(scanner.Text()), "fatal") {
			return errors.New(scanner.Text())
		}

		clio.Info(scanner.Text())
	}
	clio.Debugf("Successfully cloned %s", repoURL)

	// If a ref is specified, checkout that ref
	if ref != "" {
		clio.Debugf("git -C %s checkout %s\n", repoDirPath, ref)
		checkoutCmd := exec.Command("git", "-C", repoDirPath, "checkout", ref)

		stderr, _ := checkoutCmd.StderrPipe()
		if err := checkoutCmd.Start(); err != nil {
			return err
		}

		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			if strings.Contains(strings.ToLower(scanner.Text()), "error") || strings.Contains(strings.ToLower(scanner.Text()), "fatal") {
				return errors.New(scanner.Text())
			}
			clio.Info(scanner.Text())
		}
		clio.Debugf("Successfully checked out %s", ref)
	}

	return nil
}

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

	return nil
}

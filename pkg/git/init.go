package git

import (
	"os/exec"

	"github.com/common-fate/clio"
)

func Init(repoDirPath string) error {
	clio.Debugf("git init %s\n", repoDirPath)

	cmd := exec.Command("git", "init", repoDirPath)

	err := cmd.Run()
	if err != nil {
		return err

	}

	return nil
}

//go:build windows

package forkprocess

import (
	"os/exec"

	"github.com/pkg/errors"
)

type Process struct {
	UID     uint32
	GID     uint32
	Args    []string
	Workdir string
}

// New creates a new Process with the current user's user and group ID.
// Call Start() on the returned process to actually it.
func New(args ...string) (*Process, error) {
	p := Process{
		Args: args,
	}
	return &p, nil
}

// Start launches a detached process.
// In Windows we fall back to exec.Command().
func (p *Process) Start() error {
	cmd := exec.Command(p.Args[0], p.Args[1:]...)
	err := cmd.Start()
	if err != nil {
		return errors.Wrap(err, "starting command")
	}
	// detach from this new process because it continues to run
	return cmd.Process.Release()
}

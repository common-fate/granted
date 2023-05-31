//go:build !windows

// Package forkprocess starts a process which runs in the background.
// In Granted we use it to launch a browser when the user requests a web console.
// Previously, we used exec.Command from Go's stdlib for this, but is susceptible
// to being closed when the user pressed CTRL+C in their terminal.
//
// Thanks to @patricksanders for the advice here.
// The github.com/ik5/fork_process package is also a good
// reference we'd like to acknowledge.
package forkprocess

import (
	"os"
	"os/user"
	"strconv"
	"syscall"

	"github.com/pkg/errors"
)

type Process struct {
	UID     uint32
	GID     uint32
	Args    []string
	Workdir string
}

// New creates a new Process with the current user's user and group ID.
// Call Start() on the returned process to actually start it.
func New(args ...string) (*Process, error) {
	u, err := user.Current()
	if err != nil {
		return nil, errors.Wrap(err, "getting current user")
	}
	uid, err := strconv.ParseUint(u.Uid, 10, 32)
	if err != nil {
		return nil, errors.Wrapf(err, "parsing uid (%s)", u.Uid)
	}
	gid, err := strconv.ParseUint(u.Gid, 10, 32)
	if err != nil {
		return nil, errors.Wrapf(err, "parsing gid (%s)", u.Uid)
	}

	p := Process{
		UID:  uint32(uid),
		GID:  uint32(gid),
		Args: args,
	}
	return &p, nil
}

// Start launches a detached process under the current user and group ID.
func (p *Process) Start() error {
	var cred = &syscall.Credential{
		Uid:         p.UID,
		Gid:         p.GID,
		NoSetGroups: true,
	}

	var sysproc = &syscall.SysProcAttr{
		Credential: cred,
		Setsid:     true,
	}

	rpipe, wpipe, err := os.Pipe()
	if err != nil {
		return errors.Wrap(err, "getting read and write files")
	}
	defer rpipe.Close()
	defer wpipe.Close()

	attr := os.ProcAttr{
		Dir: p.Workdir,
		Env: os.Environ(),
		Files: []*os.File{
			rpipe,
			wpipe,
			wpipe,
		},
		Sys: sysproc,
	}
	process, err := os.StartProcess(p.Args[0], p.Args, &attr)
	if err != nil {
		return errors.Wrap(err, "starting process")
	}

	err = process.Release()
	if err != nil {
		return errors.Wrap(err, "releasing process")
	}
	return nil
}

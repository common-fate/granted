package alias

import "fmt"

type ErrShellNotSupported struct {
	Shell string
}

func (e *ErrShellNotSupported) Error() string {
	return fmt.Sprintf("unsupported shell %s", e.Shell)
}

type ErrAlreadyInstalled struct {
	File string
}

func (e *ErrAlreadyInstalled) Error() string {
	return fmt.Sprintf("the Granted alias has already been added to %s", e.File)
}

type ErrNotInstalled struct {
	File string
}

func (e *ErrNotInstalled) Error() string {
	return fmt.Sprintf("the Granted alias hasn't been added to %s", e.File)
}

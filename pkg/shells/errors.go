package shells

import "fmt"

type ErrLineAlreadyExists struct {
	File string
}

func (e *ErrLineAlreadyExists) Error() string {
	return fmt.Sprintf("the line has already been added to %s", e.File)
}

type ErrLineNotFound struct {
	File string
}

func (e *ErrLineNotFound) Error() string {
	return fmt.Sprintf("the line was not found in file %s", e.File)
}

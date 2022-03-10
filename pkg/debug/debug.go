package debug

import (
	"fmt"
	"io"
)

// This package provides a lightweight way to set global debugging verbosity for the cli, if we need structured logging or more features, consider using an existing library
// To use, add a cli flag that sets the value of CliVerbosity to one of the supported values in the enum
// e.g debug.CliVerbosity = debug.VerbosityDebug
// For convenience, you can add debug output using debug.Fpringf(debug.VerbosityDebug, color.Error, "my error output: %s", err.Error())

type Verbosity int

//go:generate go run github.com/alvaroloes/enumer -type=Verbosity -linecomment
const (
	VerbosityInfo  Verbosity = iota //INFO
	VerbosityDebug                  //DEBUG
)

var CliVerbosity Verbosity = VerbosityInfo

// Will print this log with a verbosity prefix prefix using fmt.Fprintf if the specified verbosity matches the current setting
func Fprintf(verbosityLevel Verbosity, w io.Writer, format string, a ...interface{}) (int, error) {
	if verbosityLevel == CliVerbosity {
		return fmt.Fprintf(w, CliVerbosity.String()+": "+format, a...)
	}
	return 0, nil
}

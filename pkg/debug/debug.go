package debug

import (
	"fmt"
	"io"
)

// This package provides a lightweight way to set global debugging verbosity for the cli
// To use, add a cli flag that sets the value of CliVerbosity to one of the supported values in the enum
// e.g debug.CliVerbosity = debug.VerbosityDebug
// For convenience, you can add debug output using debug.Fpringf(debug.VerbosityDebug, os.Stderr, "my error output: %s", err.Error())

type Verbosity int

const (
	VerbosityDefault Verbosity = iota //0
	VerbosityDebug                    //1
)

var CliVerbosity Verbosity = VerbosityDefault

// Will print this log with a "DEBUG: " prefix using fmt.Fprintf if the specified verbosity matches the current setting
func Fprintf(verbosityLevel Verbosity, w io.Writer, format string, a ...interface{}) (int, error) {
	debugPrefix := "DEBUG: "
	if verbosityLevel == CliVerbosity {
		return fmt.Fprintf(w, debugPrefix+format, a...)
	}
	return 0, nil
}

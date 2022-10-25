package debug

import "os"

var Debug bool

// Enable enables debug logging
//
// It sets the environment variable GRANTED_LOG=debug
func Enable() {
	Debug = true
	// CLIO debug logging is configured via this environment variable
	// setting it to "debug" means that clio.Debug() logs will be printed
	os.Setenv("GRANTED_LOG", "debug")
}

// Disable disabled debug logging
//
// It unsets the environment variable GRANTED_LOG
func Disable() {
	Debug = false
	os.Unsetenv("GRANTED_LOG")
}

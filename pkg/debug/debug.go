package debug

import (
	"github.com/common-fate/clio"
)

var Debug bool

// Enable enables debug logging
func Enable() {
	Debug = true
	clio.SetLevelFromString("debug")
}

// Disable disabled debug logging
//
// It unsets the environment variable GRANTED_LOG
func Disable() {
	Debug = false
	clio.SetLevelFromString("")
}

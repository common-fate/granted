package assumeprint

import (
	"os"
	"runtime"
)

// SafeOutput formats a string to match the requirements of granted output in the shell script
// Currently in windows, the grantedoutput is handled differently, as linux and mac support the exec cli flag whereas windows does not yet have support
// this method may be changed in future if we implement support for "--exec" in windows
func SafeOutput(s string) string {
	// if the GRANTED_ALIAS_CONFIGURED env variable isn't set,
	// we aren't running in the context of the `assume` shell script.
	// If this is the case, don't add a prefix to the output as we don't have the
	// wrapper shell script to parse it.
	if os.Getenv("GRANTED_ALIAS_CONFIGURED") != "true" {
		return ""
	}
	out := "GrantedOutput"
	if runtime.GOOS != "windows" {
		out += "\n"
	} else {
		out += " "
	}
	return out + s
}

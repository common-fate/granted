package build

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
	BuiltBy = "unknown"
)

func IsDev() bool {
	return Version == "dev"
}

// AssumeScriptName returns the name of the shell script which wraps the assume binary
func AssumeScriptName() string {
	if IsDev() {
		return "dassume"
	}
	return "assume"
}
func AssumeBinaryName() string {
	if IsDev() {
		return "dassumego"
	}
	return "assumego"
}

func GrantedBinaryName() string {
	if IsDev() {
		return "dgranted"
	}
	return "granted"
}

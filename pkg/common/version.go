package common

// NAME of the App
var NAME = "tribaloutpost-adl"

// SUMMARY of the Version
var SUMMARY = "v1.0.0"

// BRANCH of the Version
var BRANCH = "dev"

// VERSION of Release
var VERSION = "1.0.0"

var COMMIT = "dirty"

// ADLKey is the API key sent as X-ADL-Key header, set at compile time via ldflags
var ADLKey = ""

// AppVersion --
var AppVersion AppVersionInfo

// AppVersionInfo --
type AppVersionInfo struct {
	Name    string
	Version string
	Branch  string
	Summary string
	Commit  string
}

func init() {
	AppVersion = AppVersionInfo{
		Name:    NAME,
		Version: VERSION,
		Branch:  BRANCH,
		Summary: SUMMARY,
		Commit:  COMMIT,
	}
}

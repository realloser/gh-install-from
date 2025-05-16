package version

// These variables are set during build time using -ldflags
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// Info returns version information as a string
func Info() string {
	return Version
}

// DetailedInfo returns detailed version information
func DetailedInfo() string {
	return "Version: " + Version + "\nCommit: " + Commit + "\nBuild Date: " + Date
} 
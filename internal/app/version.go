package app

// BuildInfo holds build-time information
type BuildInfo struct {
	Version   string
	BuildTime string
	CommitSHA string
}

// DefaultBuildInfo returns default build information for development
func DefaultBuildInfo() BuildInfo {
	return BuildInfo{
		Version:   "dev",
		BuildTime: "unknown",
		CommitSHA: "unknown",
	}
}

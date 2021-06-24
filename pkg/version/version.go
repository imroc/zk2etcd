package version

import (
	"fmt"
	"runtime"
)

var (
	// version is the current version of the cls.
	version = ""

	// BuildMetadata is extra build time data
	buildMetadata = "unreleased"
	// GitCommit is the git sha1
	gitCommit = ""
	// GitTreeState is the state of the git tree
	gitTreeState = ""
	buildDate    = ""
)

// getVersion returns the semver string of the version
func getVersion() string {
	if buildMetadata == "" {
		return version
	}
	return version + "+" + buildMetadata
}

type Version struct {
	Version      string
	GitCommit    string
	GitTreeState string
	GoVersion    string
	BuildDate    string
	Compiler     string
	Platform     string
}

func GetVersion() *Version {
	return &Version{
		Version:      getVersion(),
		GitCommit:    gitCommit,
		GitTreeState: gitTreeState,
		GoVersion:    runtime.Version(),
		BuildDate:    buildDate,
		Compiler:     runtime.Compiler,
		Platform:     fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

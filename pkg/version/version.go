package version

import (
	"fmt"
	"runtime"
)

var (
	gitCommit  = "none"
	gitVersion = "none"
	buildDate  = "none"
)

const (
	gtctlVersion = "0.1.0"
)

type Version struct {
	GtctlVersion string
	GitCommit    string
	GitVersion   string
	GoVersion    string
	Compiler     string
	Platform     string
	BuildDate    string
}

func (v Version) String() string {
	format := "GtctlVersion: %s\n" +
		"GitCommit: %s\n" +
		"GitVersion: %s\n" +
		"GoVersion: %s\n" +
		"Compiler: %s\n" +
		"Platform: %s\n" +
		"BuildDate: %s\n"
	return fmt.Sprintf(format, v.GtctlVersion, v.GitCommit, v.GitVersion, v.GoVersion, v.Compiler, v.Platform, v.BuildDate)
}

func Get() Version {
	return Version{
		GtctlVersion: gtctlVersion,
		GitCommit:    gitCommit,
		GitVersion:   gitVersion,
		GoVersion:    runtime.Version(),
		Compiler:     runtime.Compiler,
		Platform:     fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		BuildDate:    buildDate,
	}
}

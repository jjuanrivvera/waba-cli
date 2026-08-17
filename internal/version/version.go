// Package version carries build metadata injected via -ldflags at link time.
package version

import (
	"fmt"
	"runtime"
	"runtime/debug"
)

// Injected by the Makefile / GoReleaser via -X.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// Info is the structured form used by `version --json` and `doctor --json`.
type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	Date      string `json:"date"`
	GoVersion string `json:"go_version"`
	Platform  string `json:"platform"`
}

// Get returns the current build's metadata, falling back to Go's own build info when the
// binary was produced by `go install` rather than the Makefile (which sets no ldflags).
func Get() Info {
	i := Info{
		Version:   Version,
		Commit:    Commit,
		Date:      Date,
		GoVersion: runtime.Version(),
		Platform:  runtime.GOOS + "/" + runtime.GOARCH,
	}
	if i.Version == "dev" || i.Version == "" {
		if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
			i.Version = bi.Main.Version
		}
	}
	return i
}

// String renders a one-line summary.
func (i Info) String() string {
	return fmt.Sprintf("waba %s (commit %s, built %s, %s, %s)",
		i.Version, i.Commit, i.Date, i.GoVersion, i.Platform)
}

package version

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGet_ReportsBuildMetadata(t *testing.T) {
	info := Get()
	assert.NotEmpty(t, info.Version)
	assert.Equal(t, runtime.Version(), info.GoVersion)
	assert.Equal(t, runtime.GOOS+"/"+runtime.GOARCH, info.Platform)
}

func TestGet_UsesLdflagsWhenSet(t *testing.T) {
	// The Makefile and GoReleaser inject these; a released binary must report the tag rather
	// than "dev".
	restore := []string{Version, Commit, Date}
	Version, Commit, Date = "v1.2.3", "abc1234", "2026-07-27T00:00:00Z"
	defer func() { Version, Commit, Date = restore[0], restore[1], restore[2] }()

	info := Get()
	assert.Equal(t, "v1.2.3", info.Version)
	assert.Equal(t, "abc1234", info.Commit)
	assert.Equal(t, "2026-07-27T00:00:00Z", info.Date)
}

func TestInfo_String(t *testing.T) {
	restore := Version
	Version = "v9.9.9"
	defer func() { Version = restore }()

	got := Get().String()
	assert.Contains(t, got, "waba")
	assert.Contains(t, got, "v9.9.9")
	assert.Contains(t, got, runtime.GOOS)
}

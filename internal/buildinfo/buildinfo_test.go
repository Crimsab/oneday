package buildinfo

import (
	"strings"
	"testing"
)

func TestCurrentPrefersExplicitLdflagsValues(t *testing.T) {
	oldVersion, oldCommit, oldBuildDate, oldDirty := Version, Commit, BuildDate, Dirty
	t.Cleanup(func() {
		Version, Commit, BuildDate, Dirty = oldVersion, oldCommit, oldBuildDate, oldDirty
	})

	Version = "v9.9.9"
	Commit = "1234567890abcdef"
	BuildDate = "2026-04-09T20:00:00Z"
	Dirty = "true"

	info := Current()
	if info.Version != "v9.9.9" {
		t.Fatalf("Version = %q, want explicit ldflags value", info.Version)
	}
	if info.Commit != "1234567890abcdef" {
		t.Fatalf("Commit = %q, want explicit ldflags value", info.Commit)
	}
	if info.BuildDate != "2026-04-09T20:00:00Z" {
		t.Fatalf("BuildDate = %q, want explicit ldflags value", info.BuildDate)
	}
	if !info.Dirty {
		t.Fatal("Dirty = false, want explicit true value")
	}
}

func TestInfoShortCommitTruncatesLongHashes(t *testing.T) {
	info := Info{Commit: "1234567890abcdef"}
	if got := info.ShortCommit(); got != "1234567890ab" {
		t.Fatalf("ShortCommit() = %q, want 1234567890ab", got)
	}
}

func TestTextIncludesCoreBuildFields(t *testing.T) {
	oldVersion, oldCommit, oldBuildDate, oldDirty := Version, Commit, BuildDate, Dirty
	t.Cleanup(func() {
		Version, Commit, BuildDate, Dirty = oldVersion, oldCommit, oldBuildDate, oldDirty
	})

	Version = "v1.2.3"
	Commit = "abcdef123456"
	BuildDate = "2026-04-09T20:00:00Z"
	Dirty = "false"

	output := Text("oneday")
	for _, needle := range []string{
		"oneday v1.2.3",
		"commit: abcdef123456",
		"built: 2026-04-09T20:00:00Z",
		"dirty: false",
		"go: ",
	} {
		if !strings.Contains(output, needle) {
			t.Fatalf("Text() missing %q:\n%s", needle, output)
		}
	}
}

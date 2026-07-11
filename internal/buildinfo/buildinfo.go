package buildinfo

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"
)

// These values are stamped via -ldflags during local and release builds.
var (
	Version   = "dev"
	Commit    = ""
	BuildDate = ""
	Dirty     = ""
)

// Info describes the provenance of a built binary.
type Info struct {
	Version   string
	Commit    string
	BuildDate string
	Dirty     bool
	GoVersion string
}

// Current returns the best available build metadata, preferring ldflags and
// falling back to Go's embedded VCS settings when present.
func Current() Info {
	info := Info{
		Version:   strings.TrimSpace(Version),
		Commit:    strings.TrimSpace(Commit),
		BuildDate: strings.TrimSpace(BuildDate),
		Dirty:     parseDirtyFlag(Dirty),
		GoVersion: runtime.Version(),
	}

	if build, ok := debug.ReadBuildInfo(); ok {
		if info.GoVersion == "" {
			info.GoVersion = build.GoVersion
		}
		if info.Version == "" || info.Version == "dev" {
			if version := strings.TrimSpace(build.Main.Version); version != "" && version != "(devel)" {
				info.Version = version
			}
		}
		for _, setting := range build.Settings {
			switch setting.Key {
			case "vcs.revision":
				if info.Commit == "" {
					info.Commit = strings.TrimSpace(setting.Value)
				}
			case "vcs.time":
				if info.BuildDate == "" {
					info.BuildDate = strings.TrimSpace(setting.Value)
				}
			case "vcs.modified":
				if Dirty == "" {
					info.Dirty = parseDirtyFlag(setting.Value)
				}
			}
		}
	}

	if info.Version == "" {
		info.Version = "dev"
	}
	if info.Commit == "" {
		info.Commit = "unknown"
	}
	if info.BuildDate == "" {
		info.BuildDate = "unknown"
	}
	if info.GoVersion == "" {
		info.GoVersion = "unknown"
	}

	return info
}

// ShortCommit returns a display-friendly commit identifier.
func (i Info) ShortCommit() string {
	commit := strings.TrimSpace(i.Commit)
	switch {
	case commit == "":
		return "unknown"
	case len(commit) <= 12:
		return commit
	default:
		return commit[:12]
	}
}

// Text renders human-readable provenance for a named binary.
func Text(binaryName string) string {
	info := Current()
	name := strings.TrimSpace(binaryName)
	if name == "" {
		name = "oneday"
	}
	return strings.Join([]string{
		fmt.Sprintf("%s %s", name, info.Version),
		fmt.Sprintf("commit: %s", info.ShortCommit()),
		fmt.Sprintf("built: %s", info.BuildDate),
		fmt.Sprintf("dirty: %t", info.Dirty),
		fmt.Sprintf("go: %s", info.GoVersion),
	}, "\n")
}

func parseDirtyFlag(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "dirty":
		return true
	default:
		return false
	}
}

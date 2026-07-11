# Technology Stack

**Analysis Date:** 2026-04-09

## Languages

**Primary:**
- Go 1.25.0 - application, engine, storage, TUI, and CLI tools in `cmd/` and `internal/` (`go.mod`)

**Secondary:**
- YAML - runtime config in `config.example.yaml`, local overrides in `config.yaml`, CI in `.github/workflows/*.yml`
- Markdown - product and benchmark docs in `README.md`, `docs/`, and `.planning/`

## Runtime

**Environment:**
- Native Go CLI runtime - main app starts from `cmd/oneday/main.go`
- Terminal UI runtime - Bubble Tea alt-screen + mouse mode enabled in `cmd/oneday/main.go`

**Package Manager:**
- Go modules
- Lockfile: present (`go.sum`)

## Frameworks

**Core:**
- `github.com/charmbracelet/bubbletea` v1.3.10 - event loop and model/update/view runtime for `internal/tui/`
- `github.com/charmbracelet/bubbles` v1.0.0 - textarea, viewport, spinner, and key helpers in `internal/tui/views/` and `internal/tui/components/`
- `github.com/charmbracelet/lipgloss` v1.1.1-0.20250404203927-76690c660834 - styling and layout in `internal/tui/theme/` and `internal/tui/views/`

**Testing:**
- Go standard test runner - package tests in `internal/**/*_test.go`

**Build/Dev:**
- Go toolchain - `go build`, `go test`, `go vet`
- `Makefile` - local build/test shortcuts
- GitHub Actions - build/test/release in `.github/workflows/build-release.yml`
- Release Please - release PR/tag automation in `.github/workflows/release-please.yml`

## Key Dependencies

**Critical:**
- `modernc.org/sqlite` v1.48.1 - embedded SQLite driver used by `internal/storage/db.go`
- `github.com/google/uuid` v1.6.0 - story/session/save IDs in `internal/engine/` and `internal/storage/`
- `gopkg.in/yaml.v3` v3.0.1 - config parsing in `internal/config/config.go`

**Infrastructure:**
- `github.com/charmbracelet/glamour` v1.0.0 - markdown rendering in `internal/tui/components/markdown.go`
- `github.com/muesli/reflow` - wrapping helpers used by TUI components

## Configuration

**Environment:**
- Safe defaults live in `internal/config/config.go`
- Local runtime overrides load from `config.yaml` resolved by `cmd/oneday/main.go`
- Tracked template lives in `config.example.yaml`

**Build:**
- `Makefile`
- `.github/workflows/build-release.yml`
- `.github/workflows/release-please.yml`
- Repo-root ignored binaries `oneday`, `oneday-benchmark`, `oneday-ascii-benchmark`

## Platform Requirements

**Development:**
- Go 1.25 toolchain
- Writable data directory configured by `Config.DataDir` (`internal/config/config.go`)
- Access to at least one configured AI provider for gameplay paths

**Production:**
- Native terminal binary on Linux or Windows from `cmd/oneday/main.go`
- No long-running server or container target detected in this repo

---

*Stack analysis: 2026-04-09*

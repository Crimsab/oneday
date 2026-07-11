# Summary 37.3 - Branch-safe audio APIs and browser controls

The Go/Rust gateway now exposes provider and voice discovery, story TTS policy,
voice assignments, committed-message generation/status, and immutable audio
media. Story controls reject stale revisions. Audio lookup is restricted to the
active branch, ready status, regular files, a 64 MiB response limit, and paths
inside the configured canonical audio root.

The browser adds compact native playback to committed assistant messages and an
Options editor for global mode, separate autoplay, language, narrator,
protagonist, NPC voices, and per-character inherit/on/off policy. Disabled
providers, loading, saving, empty, blocked-autoplay, failure, and retry states
remain explicit and keyboard accessible. Concurrent messages deduplicate story
settings reads. React Scan found no render loop; its temporary instrumentation
and dependency were removed before delivery.

Validation passed with Go 77 in touched packages and the complete Go suite,
Rust 47, browser 99, production build, and Playwright 10/10 across desktop and
mobile. Live `homelab-main` checks passed on schema V35 with integrity `ok`, two
stories and 15 visual assets preserved, zero audio assets/jobs under the safe
default `off` policy, provider-unavailable UI, no horizontal overflow, and
gateway restart count zero. Backup:
`/opt/lab/backups/oneday/phase37-3-20260711`.

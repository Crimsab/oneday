# Summary 36.3 — Courtroom, comedy, and accessible selection

Courtroom and social-comedy reducers now provide zero-combat, timing-free
encounters. Courtroom play grades procedural choice plus committed evidence;
comedy play grades callbacks/leverage and conversational choice without reflex
input. Both carry deterministic seeded variation and authoritative costs,
progress, failures, or successes through the shared host.

The selection policy scores narrative tag fit and requested difficulty, rewards
player preferences, excludes disallowed/reflex families, penalizes earlier
repetition, and enforces per-family turn cooldowns. If policy leaves no valid
candidate it fails explicitly instead of silently violating accessibility.
Selection reasons are stored on the definition and shown to the player.

The gateway derives recent usage from the active branch rather than trusting a
client-supplied history. Browser Auto-fit always requests timing-free play and
supplies current story genre/tone; explicit Courtroom and Comedy launchers are
also available. The TUI recognizes both families through the same reducer
adapter.

Go 512, Rust 45, browser 96, build, and 8 desktop/mobile Playwright scenarios
passed. The live V34 deployment on `homelab-main` reports restart count zero,
SQLite integrity `ok`, no active test instance, all seven launch controls on
desktop/mobile, no console errors, and no horizontal overflow.

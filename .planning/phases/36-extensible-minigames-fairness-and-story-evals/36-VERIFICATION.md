# Phase 36 verification

## Requirements

- MINI-01–MINI-05: repaired legacy reducers, shared versioned host, six new
  cross-surface families, graded outcomes, accessibility and anti-repeat policy.
- MINI-06: typed semantic story-pack authoring for stats, difficulty profiles,
  challenge pools, cooldowns, and outcome policies.
- MINI-07: versioned 12-case corpus covering generosity bias, anti-loop,
  false identity, metamorphosis, zero-combat, social comedy, deterministic
  replay, and cross-surface parity.

## Automated evidence

- Go: 519 tests passed across 23 packages.
- Rust gateway: 46 tests passed.
- Browser: 97 tests passed; production TypeScript/Vite build passed.
- Golden eval: 12/12 cases, 6 successful outcomes, success rate 0.50, all eight
  required concern counters non-zero.
- Shared fixture: Go, Rust, and TypeScript validate `contracts/minigame-v1.json`
  and confirm authoritative answers are absent from the player contract.

## Live evidence

Deployment target: `root@192.168.50.40:/opt/lab/docker/oneday` only. Schema V34
is unchanged by Plan 36.4. Live health reports `ok` with two stories; SQLite
integrity is `ok`; the gateway is running with restart count zero; the typed
noir story pack validates; the standalone eval passes 12/12; and all eight
desktop/mobile Playwright scenarios pass on the new server. The pre-deploy
database and binary are retained in
`/opt/lab/backups/oneday/phase36-4-20260711`.

# Phase 38 verification

Status: passed on 2026-07-11.

## Requirements

- UNIV-01: bounded offscreen agency emits inspectable active-branch canonical
  events without reading or mutating private thought state.
- UNIV-02: the accessible browser map projects only player-known canonical
  locations and edges and marks the current location.
- UNIV-03: branch-specific Markdown, JSON, valid EPUB3, and path-safe media
  replay exports are available.
- UNIV-04: strict YAML authoring rejects unknown or invalid stat, world,
  difficulty, challenge, visual, and licensed voice definitions.
- UNIV-05: the universal local/CI/release matrix is enforced and green.
- UNIV-06: migration, legacy save/config/story-pack/asset compatibility suites
  are part of that mandatory gate and pass.

## Clean-checkout release evidence

- Canonical commit: `431c5a3`.
- Go: 541 tests plus vet and targeted migration/compatibility suites passed.
- Minigame evaluation: 12/12 cases passed; success rate 0.50; all required
  fairness and portability concerns covered.
- Rust gateway: 47 tests, formatting, clippy, and release build passed.
- Browser: 100 Vitest tests and production build passed.
- Playwright: 12/12 desktop/mobile scenarios passed, including minigames, map,
  bounded agency, visual lineage, audio controls, and branch history/export.
- Docker gateway build and friend-safe hygiene passed.
- Main, benchmark, ASCII benchmark, Linux, and Windows artifacts exist; main
  binary reports commit `431c5a30cb1f` and `dirty: false`.

## Live new-server evidence

- Target: `root@192.168.50.40`, `/opt/lab/docker/oneday`, container
  `oneday-gateway`, domain `oneday.homelab.local`.
- Repository clean at `431c5a3`; container running with restart count zero.
- `/api/health` returned 200 with two stories.
- Schema V35; SQLite integrity `ok`; zero foreign-key violations.
- Live data remained two stories, 15 visual assets, and zero accidental audio
  assets.
- Agency endpoint responded; TTS catalog exposed two providers; audio cleanup
  dry-run was safe.
- EPUB response was base64 `application/epub+zip` with ZIP magic; replay export
  used `oneday-replay-v1` and contained no absolute server paths.

The legacy miniPC checkout and its stopped container were not used for the
implementation, deployment, or final verification.

# Phase 35 Verification

**Status:** passed and deployed on `homelab-main`

## Automated verification

- `go test ./...`: 489 passed across 22 packages.
- `cargo fmt --check`: passed.
- `cargo test`: 44 passed, including repeated V33 schema ensure and
  branch-local version selection/undo/redo with sibling isolation.
- `bun test`: 95 passed.
- `bun run build`: TypeScript and Vite production build passed.
- `bun run test:e2e`: 6 Playwright tests passed across desktop Chromium and
  Pixel 7 responsive Chromium, including canon lineage and selection controls.

## Migration and live verification

- A fresh copy of the live V32 database migrated to V33 with `integrity_check`
  `ok`, no foreign-key violations, all six branch override fields present, and
  12 assets, 5 versions, and 5 jobs preserved.
- Created `/opt/lab/backups/oneday/phase35-20260711/oneday.db` plus
  `oneday.prephase35`; backup integrity passed.
- Live schema preflight returned `{"status":"ok"}` and `schema_version=33`.
- A first live start exposed a non-idempotent Rust index ensure after the Go
  migrator had created the same V33 index. `IF NOT EXISTS` plus a repeat-ensure
  regression test removed the restart; the final deployment reports restart
  count zero and one clean gateway-listening event.
- Health and visual asset APIs returned HTTP 200 with branch, commit, canonical
  location/entity/form, fingerprint, profile revision, gate, selection, and
  inheritance fields. Live SQLite integrity returned `ok` with no FK rows.
- Headless desktop and 412 px mobile checks opened Visual Direction with HTTP
  200, zero console/page errors, and zero horizontal overflow.

## Requirement disposition

- VISUAL2-01 through VISUAL2-05: satisfied.

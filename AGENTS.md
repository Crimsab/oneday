# OneDay Deployment Target

## Critical Host Invariant

- The active OneDay source tree and runtime are on `homelab-main` at
  `root@192.168.50.40:/opt/lab/docker/oneday`.
- The active container is `oneday-gateway` on `homelab-main`.
- `/opt/lab/docker/oneday` on the miniPC `homelab.local` is a legacy,
  powered-down deployment copy. Do not develop, test, build, or deploy OneDay
  there.
- From the miniPC, operate on the active project with:
  `ssh root@192.168.50.40`.
- Verify the target before any mutation:
  `hostname` must return `homelab-main`, and
  `docker ps --filter name=oneday` must show `oneday-gateway`.

## Project Workflow

- Follow `/workspace/homelab/AGENTS.md` and the central HomeLab conventions.
- Do not use Git unless the user explicitly asks.
- Use `bun` for frontend scripts and Go/Cargo for their native project tasks.
- Preserve `homelab_network` and `TZ=Europe/Rome` in compose changes.
- After source changes, run the relevant Go, Rust gateway, and frontend tests
  on `homelab-main`; rebuild/restart the live container only when the active
  implementation slice requires deployment verification.


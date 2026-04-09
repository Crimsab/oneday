# Summary 15.3: Failure Surfacing and Deterministic Recovery

Critical canonical turn persistence failures are no longer silently swallowed in the main narrative loop. On canonical commit failure, the narrator restores its in-memory story/character/world snapshots and aborts the turn instead of leaving local state advanced while persistence failed.

Mirror-only JSONL failures are surfaced as degraded-mode warnings rather than fatal turn corruption. Achievement persistence and NPC `last_seen` updates now happen only after the canonical turn commit succeeds, and narrator-managed setting/world mutations are applied in memory first so world-state ownership stays with the canonical commit path.

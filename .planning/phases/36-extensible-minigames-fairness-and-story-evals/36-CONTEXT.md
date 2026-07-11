# Phase 36 Context — Extensible minigames, fairness, and story evals

Phase 32 established the authoritative `challenge.v1` outcome envelope, but
legacy active minigames still had TUI-only state and boolean-biased scoring.
Phase 36 adds a serializable reducer host used by every surface, repairs legacy
mechanics, expands genre-neutral challenge families, then gates them with
selection policy, story-pack authoring data, and deterministic evals.

All mechanics must remain seed/input replayable. Reflex input is optional and
must always have a timing-free accessibility alternative. Story packs provide
framing, pools, stats, difficulty profiles, and outcome policy; the engine must
not hardcode fantasy, combat, or human anatomy assumptions.

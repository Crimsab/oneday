# Summary 37.1 - Canonical audio storage

Schema V35 adds story generation/autoplay policy, versioned provider voice
profiles, entity/identity/form assignments, pronunciation entries, complete
cache identities, branch-scoped audio assets, and durable TTS jobs. JSON fields,
status enums, speed/format ranges, foreign keys, and immutable lineage are
enforced by SQLite.

Major cast cannot share an exact voice profile unless the assignment records an
explicit duplicate override. Locked assignments reject later voice changes.
Aliases can share the entity assignment, while identity and form overrides have
stable assignment keys. Settings default to `off`, so migration never generates
audio for existing stories.

Typed storage APIs cover settings, profiles, assignments, and pronunciation
revisions. Fresh tests and a V34-to-V35 preflight passed. Live migration on
`homelab-main` preserved 15 visual assets and two stories, reports SQLite
integrity `ok`, zero foreign-key violations, health `ok`, and restart count zero.
Backup: `/opt/lab/backups/oneday/phase37-1-20260711`.

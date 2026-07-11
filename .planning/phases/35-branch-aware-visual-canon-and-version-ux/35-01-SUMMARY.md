# Summary 35.1 — Visual lineage schema and reachable catalog

Migration V33 replaces the subject-collapsing visual asset identity with a
branch-local lineage key and adds canonical entity, canonical location, form,
appearance fingerprint, profile revision, canon status, and gating fields.
Versions and jobs carry the same causal identity. Versioned visual profiles,
branch prompt overrides, and branch selection state now have dedicated schema.

The migration rebuilds `visual_assets` without deleting child versions/jobs or
generated files, backfills legacy rows, reenables foreign keys, and fails on any
foreign-key violation. A copy of the live schema V32 database migrated to V33
with integrity `ok`, zero foreign-key violations, and all 12 assets, 5 versions,
and 5 jobs preserved.

Gateway catalog, version, profile, and ownership reads now follow active-head
commit ancestry and exclude sibling/future rows, including the same-fork-commit
timestamp edge. Profile edits create fingerprinted branch revisions rather than
overwriting the effective profile used by every branch.

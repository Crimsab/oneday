# Summary 35.3 — Branch-safe visual version controls

Visual asset reads now resolve only versions reachable from the active branch
and fork point. A branch can choose a version without mutating its ancestor or
sibling, and each branch retains a cursor-backed selection history for undo and
redo. Prompt, gate, and operational state overrides are likewise branch-local.

The image worker records the claimed branch, source commit, canonical form,
appearance fingerprint, and profile revision on every version. Publication
rechecks that lineage before making the version selectable. The active catalog
prefers a changed or contradictory current form over a stale ready portrait, so
old imagery is suppressed while new canon is unresolved.

The browser exposes gate reason, canon status, profile revision, form lineage,
inheritance, version metadata, regenerate/select actions, and branch-local
undo/redo. Canonical gating controls whether regeneration is available and
labels silhouette generation explicitly. Responsive Playwright coverage checks
the controls on desktop and mobile.

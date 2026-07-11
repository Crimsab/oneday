# Phase 35 Context — Branch-aware visual canon and version UX

The current visual registry records branch and source commit opportunistically,
but identifies assets by story/type/subject. That collapses sibling branches,
new canonical forms, appearance changes, and profile revisions into one mutable
row. The worker also resolves the current asset after claiming an older job, so
a checkout during generation can publish an image into the wrong timeline.

This phase makes generated imagery a derived view of reachable canonical facts.
Profile revisions and asset lineages are fingerprinted; versions/jobs carry the
canonical entity, location, form, branch, source commit, and profile revision;
queries expose only ancestors of the active head. Gating remains explicit and
player-readable, and branch-local selection state provides safe select,
undo, and redo without rewriting sibling history.

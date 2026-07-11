# Summary 35.2 — Canonical gating and checkout-safe generation

Portrait candidates now expose explicit states for insufficient observation,
optional silhouette, identified draft, established canonical appearance, form
change, and identity contradiction. Canonical entity forms and player-known
appearance facts determine fingerprints; a changed form creates a separate
lineage, while unresolved identity contradictions block generation. Silhouettes
require an explicit opt-in and use only observed outline facts.

Location candidates require a canonical location ID and one of: a player-known
canonical world event, a chapter boundary, a meaningful stay, or an explicit
asset request. Automatic generation ignores blocked and explicit-only rows.
Branch-local gate overrides keep evolving eligibility from mutating an ancestor
or sibling branch.

The worker claims jobs only from the active branch whose source commit remains
reachable. It rechecks branch, commit, appearance fingerprint, and profile
revision before publication; stale output is deleted and the job is cancelled.
Active-job uniqueness is now per asset and branch, and browser-facing status no
longer reports a sibling branch's queued/running job.

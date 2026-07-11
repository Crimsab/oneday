# Plan 20.1 Summary

Added a canonical investigation-board model to world state persistence, covering cases, clues, suspects, claims, contradictions, leads, theories, links, and hidden truths as structured state instead of scattered hook prose.

Wired the board through storage migrations, world-state save/load, and rollback restore so mysteries now survive snapshot operations with hidden truths kept separate from discovered evidence.

Code commit: `64e8476 feat(investigation): add canonical board persistence`

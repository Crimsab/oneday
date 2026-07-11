---
phase: 30-universal-character-canon-and-reputation-ledgers
requirements: [CANON-01, CANON-02, CANON-03, CANON-04, CANON-05, CANON-06, CANON-07, CANON-08]
status: active
---

# Phase 30 Context

Canonical entity identity is distinct from every claim about who that entity
appears to be. Forms and controller events represent bodies, transformations,
possession, and body theft without changing the entity primary key. Facts and
reputation are append-only ledgers. The old `npcs` table remains a writable
compatibility projection synchronized to a canonical entity.

Player-safe reads are deny-by-default: only public/player-visible facts and
claims visible to the requested observer may cross the projection boundary.
Manual field locks are enforced by repository merge code and database triggers.


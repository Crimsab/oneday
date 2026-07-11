# Summary 38.1 - Bounded offscreen agency

Each committed turn can emit at most two deterministic `npc.agency` events for
living NPCs away from the current scene. A three-turn per-entity cooldown and
hard cap of three prevent runaway simulation. Events carry entity, branch,
commit, turn, action, public summary, and bounded/protocol markers; they do not
read or mutate private thoughts. The active-branch API and browser feed make the
result inspectable. Unit coverage proves bounds, cooldown, action classes, and
immutable lineage.

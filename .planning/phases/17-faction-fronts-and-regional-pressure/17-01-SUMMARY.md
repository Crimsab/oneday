# Plan 17.1 Summary

Implemented canonical faction-front state in `world_state.fronts_json`, including migration `v13`, storage round-trips, save/load rollback support, and engine-owned `front_add`, `front_advance`, `front_reveal`, `front_stall`, `front_resolve`, and `front_pressure` events.

Added tracker-facing front rendering and regression coverage so hidden fronts stay hidden until revealed and canonical front state survives save/load.

Code commit: `73e98cf feat(fronts): add canonical front state`

# Summary 13.1: Turn Delta and Hook Tracking

Added a canonical `turn_delta` payload to narrative metadata and resume flow so scenes can show a clear "What changed this turn?" block without relying only on prose. The world state now persists hook and reaction JSON, and gameplay state changes can open, progress, or resolve hooks plus attach visible world reactions that survive save/load and rollback.

The state summary fed back into the narrator now includes active hooks and world reactions, so continuity has a lightweight memory of promises, mysteries, timers, rumors, and other unresolved threads.

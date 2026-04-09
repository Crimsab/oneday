# Plan 21.1 Summary

Added a canonical project-clock board to world state persistence, covering long-arc training, rituals, crafting lines, relationship arcs, and base upgrades as structured project data instead of one-off downtime prose.

Wired project clocks through storage migrations, save/load, and rollback restore so progress survives snapshots and resumes without drifting from the rest of world state.

Code commit: `2786af1 feat(projects): add canonical downtime clock persistence`

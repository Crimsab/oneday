# Summary 13.2: Nearby NPC Conversations and World Reactivity

Expanded NPC persistence beyond disposition with multi-axis relationship JSON (`trust`, `fear`, `debt`, `respect`, `intimacy`) and surfaced those values in both narrator context and the player-facing relationship sheet. Added `/talk` as a scoped nearby-NPC conversation flow that lets the player lock onto a recent NPC with an interaction intent such as `ask`, `probe`, `bond`, `promise`, or `threaten`.

The world can now persist visible reactions and fail-forward setbacks through canonical state changes, and those consequences are folded into turn deltas, the tracker overlay, and future narrator context.

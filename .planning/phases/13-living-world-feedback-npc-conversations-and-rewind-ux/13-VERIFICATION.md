# Verification 13

## Automated Checks

- `go test ./...`
- `go build ./cmd/oneday`
- `go vet ./...`

## Verified Outcomes

- Narrative turns can persist and replay a structured `turn_delta` payload for "What changed this turn?" instead of depending entirely on prose interpretation.
- Hook tracking and world reactions are persisted canonically on `world_state`, survive save/load and rollback, and are visible through the tracker overlay and narrator context.
- NPCs now support richer relationship axes beyond disposition, and those axes are shown in context/player inspection.
- `/talk` provides a nearby-NPC scoped conversation mode with intent framing instead of dumping the whole NPC roster into the UI.
- Fail-forward/world-reaction consequences can be emitted canonically through gameplay state changes and remain visible after the causative turn.
- Saves created after loading an older snapshot now carry rewind-branch metadata, and the save picker surfaces that branch context.
- `/downtime` offers an explicit low-intensity scene request, and crafting now surfaces actionable guidance such as craftable-known recipes, near misses, material tags, and failure alternatives.

# Summary 11.1: Runtime Reliability and Turn Flow

## Delivered

- replaced rune-only typewriter slicing with ANSI-safe visible-rune playback so styled text is never cut mid-escape sequence
- stopped replaying the full rendered history on every non-streamed turn; only the new scene segment is animated now
- gated choice/input visibility until the visible scene playback is settled
- added validation plus retry/repair for story-definition generation during New Story bootstrap
- added runtime narrative-response repair before falling back to the minimal emergency narrative wrapper
- introduced a confirmation/prelude screen for active dice-roll and mini-game challenges

## Notes

- passive stat/item/skill/relationship checks still resolve automatically
- the new playback queue also keeps delayed scene additions like ambient ASCII from jumping ahead of the current scene

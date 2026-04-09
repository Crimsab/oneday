# Summary 14.3: Descriptive Codex, Dialogue Normalization, and Drill-Down Navigation

Built a canonical codex materialization layer that assembles descriptive entries for people, places, factions, mysteries, and active threads from story state, NPC records, chapters, achievements, hooks, reactions, and recent history. The new codex browser uses stacked drill-down navigation so linked entries can be opened and explored without collapsing the surrounding context.

Dialogue rendering now normalizes repeated prose-versus-`dialogue_blocks` output more safely: quoted lines repeated in the narrative blob are stripped before render, structured dialogue blocks are preserved, and a fallback extractor can still surface recognizable speaker-tagged single-quoted speech as dialogue when structured blocks are absent.

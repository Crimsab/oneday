# Feature tour

OneDay is an any-story engine rather than a genre-specific RPG. A story may be
a mystery, romance, comedy, political drama, horror tale, slice of life,
science-fiction expedition, fantasy campaign, or a form you invent. Each story
owns its language, tone, rules, visual direction, and whether combat exists.

## Write, choose, and branch

- Write any action in natural language or select a suggested choice.
- Keep multiple stories and sessions in one canonical SQLite database.
- Save, rewind, fork a decision, and explore alternate branches without
  rewriting the original timeline.
- Search story history and inspect the events that produced current state.
- Play through the React browser interface or the terminal client against the
  same engine and data.

## A world that persists

OneDay tracks more than the transcript:

- characters, motives, private thoughts, relationships, reputation, and traits;
- locations, routes, factions, fronts, hooks, pressure clocks, and fallout;
- inventory, currency, skills, recipes, crafted items, and rewards;
- clues, suspects, contradictions, investigations, and case progress;
- downtime projects, achievements, chapter history, and world events.

The model proposes narrative changes. The engine validates and commits the
structured consequences so browser and terminal never invent separate state.

## Browser workspace

- The navigation rail expands, collapses, or hides while remembering only the
  last visible desktop mode. Mobile navigation remains an independent drawer.
- Story Library searches active or archived summaries and keeps create, edit,
  archive, restore, and guarded deletion out of the navigation rail. Its detail
  pane loads overview, branches, chapters, saves, timeline, and visual assets
  only when selected; cards never fetch full story snapshots.
- The browser exports and imports a versioned UI theme containing only visual
  preferences. JSON remains the default format; an optional ZIP can include
  user-authorized WOFF2 fonts with hash verification, extraction limits, and an
  IndexedDB recovery journal. Import shows a diff, warns about missing local
  fonts, and offers one-step undo without including stories, provider settings,
  or gameplay data.
- Supported desktop Chrome browsers expose optional per-message reading
  translation below completed transcript messages. It uses the browser's
  built-in Translator API and never adds a model call or rewrites saved text.

## Interaction that fits the scene

Challenges can use stats, skills, items, relationships, dice, or story-specific
rules. Outcomes support partial success and fail-forward consequences instead
of reducing every uncertain action to pass/fail.

The engine can automatically select a contextual, timing-free minigame based on
the scene and recent branch history. Built-in families include deduction,
negotiation, pattern solving, bidding, courtroom exchanges, and comedy.
Reusable definitions also support riddles, memory, rock-paper-scissors, and
quick-time mechanics; accessibility policy can require timing-free choices.

Crafting, investigations, projects, social duels, and optional combat are
available as independent systems. A quiet relationship story does not need hit
points, while a tactical adventure can enable them.

## Generated media that does not own canon

- Scene and location art, stable character portraits, map backgrounds, and
  transparent location symbols.
- Story-specific visual direction with saved positive and negative guidance.
- Ambient ASCII art for terminal presentation.
- Spoken narration and character voices with cloud or local TTS.
- Provider routing, ordered fallbacks, retryable background jobs, cache, and
  branch-aware asset lineage.
- Manual PNG, JPEG, and static WebP uploads become additional branch-aware
  versions. Selection is explicit, and later generation preserves a selected
  manual upload.
- A manual upload may also create a new custom, world, location, or character
  asset on the active branch; it is stored as a draft-canon user asset and is
  never treated as generated canonical evidence.

Media work is asynchronous and non-blocking. Text, map structure, labels, and
story state remain authoritative when an image or audio provider is unavailable.

## Memory, providers, and operation

- RAG summaries and embeddings keep long-running stories grounded with remote
  or local embedding providers.
- Narrative generation supports Codex CLI, Claude Code, LiteLLM-compatible
  endpoints, and OpenRouter with ordered fallback.
- Visual generation supports Codex OAuth through imagegen-bridge, direct vendor
  adapters, legacy OpenClaw bridge calls, and compatible image endpoints.
- Redacted lineage stays in SQLite; optional OpenTelemetry export connects to
  Langfuse, Tempo, Jaeger, or another OTLP receiver.
- Story packs define reusable genre mechanics, stat schemas, world rules,
  challenge pools, outcome policy, visual direction, and voice direction.

Continue with [Story systems](story-systems.md), [Generated media](media.md), or
[Extensions](extensions.md) for the implementation contracts behind these
features.

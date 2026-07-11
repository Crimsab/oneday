# Automatic Challenges, Structured Settings, and Illustrated Canon Map

Date: 2026-07-11
Status: implemented and validated
Target: `root@192.168.50.40:/opt/lab/docker/oneday`

Automatic challenge selection, the searchable settings workspace, and
canonical illustrated map layers were completed together. The original manual
browser minigame chooser has been removed.

## Objective

Correct three player-facing problems without performing the later full visual
redesign:

1. Minigames must be triggered by narrative events and selected automatically.
   The player must never choose the minigame family.
2. Options must become a large, searchable settings workspace with a persistent
   category sidebar instead of one long scrolling form.
3. The known-location map must keep its canonical SVG interaction layer while
   gaining an optional ImageGen art layer and generated location symbols.

The existing visual language, routes, data, and branch semantics remain stable.

## Current-state findings

- Ordinary actions already receive an engine-owned deterministic
  `OutcomeEnvelope` before narration.
- `NarrativeResponse.challenges` still exists and the TUI can automatically
  open a challenge returned by the narrator.
- The browser does not consume that automatic challenge flow. Its Challenge
  Host is mounted for every selected story and exposes manual family buttons.
- Automatic minigame scoring exists, but only runs after the browser calls the
  manual start endpoint.
- Options combines interface preferences, runtime status, AI routing, ImageGen,
  speech, pronunciation, visual profiles, visual jobs, versions, and cleanup in
  one scroll container.
- The map is currently an accessible SVG graph with circle nodes, line edges,
  labels, and a deterministic grid layout. It has no illustrated background.
- `BRANCH-03` was the final unchecked sibling-isolation requirement. Visual and
  audio branch isolation are now directly tested and the requirement is closed.

## Automatic challenge design

### Ownership

- The NPC/narrator decides that a scene has produced an interactive challenge.
- The NPC/narrator supplies player-facing context, stakes, difficulty, involved
  NPC, and narrative tags. It does not select the final minigame family.
- The engine selects the family using story-pack rules, narrative fit,
  difficulty, recent usage, cooldowns, and accessibility policy.
- The player supplies only the input required to play the selected challenge.
- The engine resolves the result. The narrator may describe it but cannot
  change it.

This preserves the intended division: narrative intelligence detects the
moment, deterministic game logic selects and resolves the mechanic.

### Selection inputs

The selection context will include:

- story genre and tone;
- current scene type;
- challenge description and stakes;
- involved NPC and relationship context;
- canonical location and active world threads;
- related character attributes or skills;
- authored challenge pool and difficulty profile;
- recent minigame instances on the active branch;
- cooldown turns;
- timing-free accessibility policy.

The selector maps semantic situations to candidate families:

- evidence, identity, contradiction, investigation: deduction;
- bargaining, persuasion, leverage, diplomacy: negotiation;
- sequence, mechanism, ritual, decoding: pattern;
- auction, scarce resource, competing offers: bidding;
- accusation, testimony, procedure, trial: courtroom;
- performance, banter, embarrassment, social recovery: comedy.

The score still considers difficulty fit, authored preferences, and repetition.
If no candidate exceeds the confidence threshold, the engine uses an automatic
ordinary stat or dice outcome instead of starting a random minigame.

### Lifecycle

1. The player submits a normal action.
2. The current automatic ordinary outcome remains authoritative for that
   action.
3. The narrator returns the scene and may include one interactive challenge
   intent for the next beat.
4. During the same canonical turn commit, the engine validates the intent,
   selects the minigame, creates a branch-bound instance, and publishes a
   challenge-started event.
5. Browser and TUI detect the active instance automatically.
6. The browser renders the Challenge Host only while the instance is active or
   paused. There are no family launch buttons and no `Auto-fit` button.
7. Normal narrative input is disabled while the challenge is active.
8. Pause and resume remain available because they do not change difficulty or
   selection.
9. Resolution persists the authoritative result and emits a resolved event.
10. OneDay sends a structured, idempotent challenge-result continuation to the
    narrator automatically, matching the existing TUI behavior.
11. The narrator continues the scene from the resolved result. The host then
    disappears from normal play.

Only one unresolved minigame may exist per story branch. Repeated events and
reconnects must return the same instance rather than creating another one.

### Compatibility

- Old story packs that explicitly name a minigame remain loadable. Their value
  acts as an authored pool constraint, not a player choice.
- Existing manual start API compatibility may remain for tests and operator
  tooling, but it is removed from normal browser UI.
- Existing resolved instances remain replayable.
- Active instances survive refresh, reconnect, checkout, and service restart.
- Sibling branches cannot see or resolve each other's instances.

### Failure behavior

- Invalid AI challenge intent is ignored with redacted diagnostics and the
  narrative continues normally.
- No eligible candidate falls back to an ordinary automatic outcome.
- A failed continuation remains retryable and idempotent by instance ID.
- Provider or narrator failure never loses the already resolved result.

## Structured Options design

### Shell

Options becomes a dedicated settings dialog rather than a long generic overlay:

- desktop width: up to 1280px and at most the viewport minus safe margins;
- desktop height: about 88dvh;
- fixed header with title, search, save state, and close action;
- fixed left sidebar around 220-250px;
- one content scroll area on the right;
- no nested independent scroll areas inside settings sections;
- URL deep link continues to use `overlay=options` and adds an optional section
  query parameter;
- Escape closes the dialog and focus returns to the Options trigger;
- tab order, focus rings, labels, descriptions, errors, and status messages
  remain accessible.

On narrow screens the dialog becomes full-screen. The same category list is
available through a collapsible navigation drawer at the start of the dialog.
Content remains single-column with 16px minimum form text.

### Categories

1. General
   - density;
   - font size;
   - accent;
   - stories sidebar;
   - inspector panel;
   - transcript wrapping.
2. Gameplay and accessibility
   - timing-free challenge policy;
   - minigame accessibility preferences;
   - challenge status explanation;
   - no manual family selector.
3. Spoken audio
   - off, narrator, dialogue, or all;
   - autoplay;
   - default language;
   - narrator, protagonist, and NPC voice assignments;
   - pronunciation lexicon;
   - audio retry, export, and safe cleanup.
4. Visuals and map
   - story visual profile;
   - image generation policy;
   - map art and location-symbol generation;
   - asset generation jobs;
   - versions, selection, undo, redo, prompt overrides, and cleanup.
5. AI and models
   - provider order and enablement;
   - narrative, utility, repair, ASCII, and embedding models;
   - retry and fallback routing;
   - ImageGen provider and model connection settings.
6. Advanced and system
   - live update transport;
   - capabilities and active configuration revision;
   - diagnostics and safe reload controls;
   - technical status formerly shown as the four summary tiles.

### Search

Search is client-side and operates over a typed settings registry containing:

- section ID;
- setting ID;
- label;
- short description;
- keywords and common synonyms.

Results show the matching setting and its category. Selecting a result changes
category, scrolls the setting into view, focuses its control, and briefly marks
the row. Search does not expose secret values and does not index field contents.
An empty result provides a plain explanation and a clear-search action.

### Component boundaries

The current `PanelDrawer.tsx` options implementation will be split into focused
modules under `gateway/web/src/components/settings/`:

- `SettingsDialog`;
- `SettingsSidebar`;
- `SettingsSearch`;
- `GeneralSettings`;
- `GameplaySettings`;
- `AudioSettings`;
- `VisualMapSettings`;
- `ModelSettings`;
- `AdvancedSettings`;
- `settingsRegistry`.

Existing API functions and save callbacks remain the source of truth. The
refactor must not create a second settings state or silently change persistence.

## Illustrated known-location map design

### Layer model

The map keeps three independent layers:

1. Art layer
   - an optional branch-aware generated regional background;
   - decorative terrain, mood, palette, and broad geography;
   - no baked labels and no authoritative routes.
2. Canon layer
   - accessible SVG or HTML overlay;
   - only known canonical locations and edges;
   - deterministic positions, current-location state, routes, labels, keyboard
     navigation, and tooltips.
3. Symbol layer
   - generated or library-backed location symbols;
   - house, inn, settlement, ruin, forest, port, institution, and other
     authored location kinds;
   - generated at 256x256 with transparent background, then displayed at a
     smaller responsive size.

The art layer is never a source of game truth. If it invents a road, building,
or label, that detail has no mechanical or canonical meaning.

### Generation policy

- SVG-only mode remains a complete fallback.
- Automatic map art requires a configured image provider and an enabled story
  visual profile.
- Initial art generation waits until enough known geography exists to avoid an
  empty or misleading map.
- Regeneration occurs at controlled milestones, such as a new region or chapter
  with substantial graph change, not after every discovered room.
- A map fingerprint hashes branch, known location IDs, known edge IDs, region,
  and visual-profile revision.
- Existing art remains visible while a new revision is queued.
- Checkout selects only map art reachable from the active branch and commit.
- Failed generation leaves the canonical SVG map usable.

### Positioning

Location and edge placement stays deterministic and belongs to the canon layer.
The first implementation uses stable graph layout positions over a thematic
background. It does not ask ImageGen to place clickable landmarks at exact
pixels because generated images cannot guarantee geometric alignment.

Future story packs may author coordinates or region anchors. The data contract
will allow that extension without making it a requirement now.

## Testing and acceptance

### Automatic challenges

- AI intent creates one automatically selected instance without a browser start
  request.
- No manual family buttons are present.
- No host is rendered with no active instance.
- Semantic contexts choose expected families under deterministic fixtures.
- Low-confidence context falls back without a random minigame.
- Cooldown and branch isolation are enforced.
- Refresh and reconnect restore the same active instance.
- Resolution continues narration exactly once.
- Browser and TUI follow the same lifecycle.

### Options

- All current settings remain reachable in exactly one category.
- Search finds labels and synonyms and focuses the correct control.
- Desktop sidebar remains fixed while only content scrolls.
- Mobile drawer and content do not overflow horizontally.
- Keyboard navigation, Escape, focus return, errors, loading, and save states
  pass accessibility checks.
- No secret field value is included in the search index or DOM diagnostics.

### Map

- Unknown locations and edges never render in any layer.
- Art and symbols are branch and commit safe.
- SVG fallback works when ImageGen is disabled or fails.
- Keyboard and screen-reader representations remain complete.
- Generated map and icon assets do not shift layout while loading.
- Desktop and mobile rendering has no clipping or overflow.

### Release verification

- Go tests and vet;
- migration and compatibility tests;
- Rust format, test, clippy, and release build;
- browser unit tests and production build;
- Playwright desktop and mobile flows;
- minigame eval corpus;
- Docker gateway build;
- clean release provenance;
- copied-live-DB preflight and live `homelab-main` verification.

## Non-goals

- No complete visual redesign in this tranche.
- No player-facing minigame chooser.
- No AI-controlled outcome or bypass of engine resolution.
- No generated image treated as canonical geography.
- No new public route hierarchy for settings.
- No removal of legacy story, save, TUI, or story-pack compatibility.

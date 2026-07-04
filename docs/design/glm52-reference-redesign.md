# GLM-5.2 Reference Redesign (Reference 02 Source of Truth)

Extraction taken directly from `oneday-reference-pack/PRIMARY-reference-02.png`
via zai-vision image analysis. This is an image-to-code target, not inspiration.

## Source of truth

- Primary: `PRIMARY-reference-02.png` (amber, warm, cinematic).
- Variants `reference-01..10` describe the same system in alternate accents
  (e.g. reference-03 is the green-accent variant) and light/dark alternatives.
- The previous `spike/glm52-redesign` branch was discarded: it drifted into a
  compressed, terminal-only look. Do not recreate it.

## Layout (desktop, must fit first viewport)

- Three columns, no page scroll on desktop.
  - Left rail: ~20% (target ~280-300px). Brand aligned with rail, story search,
    story cards, module nav, story notes.
  - Center stage: ~55%. Bordered main reading panel dominates.
  - Right inspector: ~25% (target ~380-440px). Supportive, not cramped.
- Top bar: compact, centered status cells, right action buttons.
- Image banner + transcript dominate. Choices secondary but obvious.

## Color tokens (amber direction)

- App background: very dark charcoal, near-black with a faint warm tint.
- Panel surfaces: slightly lighter charcoal; left/right rails darkest.
- Borders: subtle 1px hairlines, low contrast (not bright).
- Accent: warm amber/gold. Used for brand, active states, execute button,
  condition heartbeat, relationship fills, number circles.
- Text: warm off-white primary, muted warm gray secondary, faint for meta.
- No AI purple. No neon glow. Matte, not glossy.

## Top bar

- Centered status cells, each `LABEL value` with small icon, separated by thin
  vertical hairlines (not dots): Turn, Day/Time, Location, Weather, Condition.
- Values readable, not cramped. Location cell flexible width.
- Right side: toggle rails, Saves, Options, Help.

## Center main panel

- Bordered, rounded panel. Header reads "Narrative Transcript" with controls.
- Cinematic location image banner at top of the panel: wide, ~140-160px tall,
  with a dark scrim gradient. Location title (large, sans, warm white) and a
  small status chip (e.g. "Morning" with sun icon) overlay it.
- Transcript below: spacious, readable body text; mono only for timestamps and
  turn/meta labels. Timestamps in amber.

## Choices (non-negotiable)

- Default pattern: exactly 3 broad horizontal cards in one row under the
  transcript at desktop. Not tiny vertical rows.
- Each card: prominent number circle (accent), title with enough room for long
  browser choices, metadata chips row (`INT`/`RISK`/`SCOPE`/`CERT`), and a
  short `Gain`/`Risk` outcome line. Thin border, subtle accent edge.
- Must not overflow at 1366x768. If more than 3 choices, wrap to a second row
  of 3 (keep the 3-card language). No carousel unless viewport is genuinely
  too narrow.
- No expanding popover that reflows the grid at rest.

## Composer

- Bottom of main panel. Natural-language placeholder. Mode select (Action /
  Talk / Advance / Time Skip). Prominent amber Execute button.
- Subtle command-tip line. Mono for the shortcut hints only.

## Right inspector (World State) - biggest structural change

The default inspector view (history tab) becomes a "World State" dashboard,
not a generic key/value module list:

1. Header "World State".
2. Metric tiles: 3 across - `TURN`, `TIME` (Day N, HH:MM), `CYCLE`
   (Morning/Afternoon/...). Big value, small uppercase label.
3. Location card: thumbnail (existing visual asset URL when ready, polished
   empty frame otherwise) + location name + region/type.
4. Condition card: condition label + a heartbeat/ECG-style line visual in
   accent color (CSS-only, no images).
5. NPCs: cards with name, role, and a relationship bar (track + accent fill by
   disposition/trust). Use existing NPC data; graceful empty state.
6. Current threads: compact list from fronts/hooks/investigations/guidance with
   a status dot.
7. Quick facts: short bullet lines (key contact, active front, next lead,
   updated time).

Other module tabs keep working via the existing module renderers; only the
default overview is redesigned. No raw JSON where a structured render exists
(raw state stays collapsed in overlays only).

## Non-negotiables

- Whole app fits first desktop viewport. No giant scrolling page.
- Desktop is primary; 1536x864 is the match target, 1366x768 must not break.
- Use existing generated visual asset URLs when present; polished empty media
  frame when absent. Never hardcode story images.
- lucide-react icons only; no SVG illustrations; no new heavy UI libraries.
- Do not modify Go/Rust backend contracts, save formats, or model routing.
- No em-dash characters in visible text.

## Files touched

- `gateway/web/src/components/Inspector.tsx` (World State overview).
- `gateway/web/src/components/SuggestedActions.tsx` (clean choice cards).
- `gateway/web/src/components/StoryPath.tsx` (status chip on banner).
- `gateway/web/src/styles.css` (world state, choices, top bar, polish).

## QA targets

- 1536x864 visually close to PRIMARY-reference-02.png.
- 1366x768 shows left rail, top bar, banner+transcript, 3 choice cards,
  composer, right inspector without broken overflow.
- Empty media frame is polished; banner fills when a URL is available.

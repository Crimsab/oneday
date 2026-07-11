# Slash Command Autocomplete And Smart Talk Design

## Goal

Make slash commands easier to discover and faster to use in the narrative TUI, with special support for `/talk`, while keeping the terminal flow lightweight and non-intrusive.

## Scope

- Show a filtered dropdown only when the free-input textarea starts with `/`.
- Support command suggestions with short descriptions.
- Support two-step `/talk` completion:
  - first nearby NPC suggestions
  - then intent suggestions
- Support `/talk` one-shot messages in the same command.
- Add single-line free-input history on `Up` / `Down`.
- Add a fast talk-mode close shortcut on `Ctrl+Space`.
- Keep normal enter-to-send behavior when the user is not explicitly accepting a suggestion.
- Treat the last crafting choice as a local exit action, in addition to text-based exit detection.

## Nearby NPC Rule

For this iteration, "nearby" means "scene-relevant now", not strict spatial simulation.

Selection order:

1. Use `engine.NearbyNPCs(...)`.
2. That function already prefers living NPCs seen in the last 3 turns.
3. If no recent NPCs exist, it falls back to the most recently seen living roster.

This keeps the feature useful now and leaves room for a future explicit scene-presence roster.

## UX

### Slash dropdown

- Hidden unless input begins with `/`.
- Filter suggestions as the user types.
- `Tab` accepts the highlighted suggestion.
- `Esc` closes the suggestion dropdown without clearing the textarea.
- The suggestion UI stays lightweight and unbordered to avoid terminal wrap artifacts.
- `Up` / `Down` are reserved for free-input history.

### Talk flow

- `/` shows all commands.
- `/ta` narrows to `/talk`.
- Accepting `/talk` inserts `/talk ` and immediately opens nearby NPC suggestions.
- Accepting an NPC inserts `/talk <NPC> ` and immediately opens intent suggestions.
- Accepting an intent inserts `/talk <NPC> <intent>`.
- `/talk <NPC> [intent] <message...>` sends a one-shot line immediately and does not enable persistent talk mode.
- `Ctrl+Space` closes persistent talk mode instantly.

### Free-input history

- Applies to the main gameplay free-input textarea.
- Stores both slash commands and normal player text.
- `Up` / `Down` browse history when the current textarea content is single-line.
- Returning past the newest history item restores the current draft.

### Crafting flow

- The final AI-provided choice is always treated as a local "leave crafting" action.
- Existing exit keyword detection remains as a fallback for textual variants such as "back" or "torna indietro".

## Architecture

- Add a small reusable suggestion model under `internal/tui/components/`.
- Keep narrative-specific suggestion assembly in `internal/tui/views/`.
- Do not change command parsing rules in the engine beyond reusing the existing registry and talk helpers.

## Testing

- Suggestion filtering for slash commands.
- `/talk` NPC and intent suggestion transitions.
- `/talk` one-shot parsing.
- Free-input history recall and draft restore.
- Accepting a suggestion updates the textarea as expected.
- Crafting last-choice exit closes locally without sending another crafting request.

## Notes

- This intentionally avoids a heavyweight command palette.
- This intentionally avoids showing suggestions when the input does not start with `/`.
- Future upgrade path: replace recency-only NPC discovery with explicit scene-presence metadata.

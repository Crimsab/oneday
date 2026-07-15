# Localization

OneDay currently ships its interface in English (`en`) and Italian (`it`).
Localization applies to controls, help, status messages, accessibility text,
validation context, and other presentation. It does not rewrite saved stories or
machine contracts.

## Three independent language settings

Keep these concepts separate when adding or changing a feature:

| Setting | Owner | Examples |
| --- | --- | --- |
| Interface locale | Browser preferences or `interface.locale` in `config.yaml` | Buttons, menus, help, errors, dates, counts |
| Story language | Canonical story state | Generated narration, choices, story exports |
| Spoken-audio language | Per-story TTS settings and voice assignments | Voice discovery, pronunciation, synthesis |

Changing the interface locale must not mutate `Story.Language`, a story-creator
definition, `default_language_tag`, a voice assignment, or stored story content.
Provider IDs, model names, slash-command tokens, JSON field names, database
enums, telemetry keys, and operational logs also remain stable.

## Locale resolution and normalization

Only the language portion is significant for the supported locales. `en-US`,
`en_GB`, and other English variants normalize to `en`; `it-IT`, `it_CH`, and
other Italian variants normalize to `it`. Unsupported or malformed values fall
back to English.

The browser resolves locale in this order:

1. the saved browser preference;
2. `navigator.languages` / the browser locale;
3. English.

The terminal resolves locale in this order:

1. `interface.locale` in `config.yaml`;
2. `LC_ALL`, `LC_MESSAGES`, then `LANG`;
3. English.

Use `oneday config locale en`, `oneday config locale it`, or
`oneday config locale auto` to update the terminal preference. Browser users can
switch languages immediately under **Options > General > Interface language**.

## Catalog structure

The React client initializes i18next in `gateway/web/src/i18n.ts`. Bundled
English and Italian resources live under `gateway/web/src/locales/` and are
divided into presentation namespaces such as `common`, `options`, `chrome`,
`story`, `branches`, `audio`, `onboarding`, and `commands`. Use the most specific
existing namespace; introduce another coherent namespace before allowing an
unrelated section to become a catch-all.

Go uses the owned boundary in `internal/i18n/`. English is the source catalog,
Italian has matching keys, and callers receive a `Localizer` instead of relying
on mutable process-global language. CLI/TUI components should translate at the
presentation boundary and preserve wrapped error causes for diagnostics.

Rust gateway responses use stable semantic codes and structured arguments for
browser presentation. Keep compatibility prose where a wire contract already
contains it, but React should render the localized code. Unknown internal
failures must use a safe localized generic message rather than expose internal
details. Story exports are the exception: their human-readable chrome follows
the saved story language, not the current interface locale.

## Adding or changing messages

1. Add the English source message to the appropriate namespace/catalog.
2. Add a natural Italian translation with the same interpolation variables and
   plural forms. Translate meaning and tone, not English word order.
3. Replace the presentation literal at its UI boundary. Include `aria-label`,
   `title`, placeholder, empty/loading/error text, notifications, confirmations,
   and screen-reader-only copy.
4. Use named interpolation for values and plural messages for counts. Do not
   build translated sentences by concatenating fragments.
5. Format user-facing dates and numbers with the active locale. Do not
   locale-format protocol values, IDs, or serialized data.
6. Add or update a focused behavior test for the affected surface.

English is the deterministic runtime fallback. Missing keys must never appear
as raw catalog identifiers in production. A missing Italian entry is a test
failure even though runtime fallback remains safe.

## Adding a locale

1. Add the normalized locale code to both the browser and Go supported-locale
   lists and resolvers.
2. Add complete React and Go catalogs with the same namespaces, keys,
   interpolation variables, and plural requirements as English.
3. Add the locale to the browser selector and terminal configuration path.
4. Extend locale normalization, resolution-precedence, catalog parity,
   fallback, interpolation, plural, and persistence tests.
5. Add desktop and mobile browser coverage and TUI render coverage for the new
   language. Exercise onboarding, story creation, a normal action, branches and
   saves, a challenge/minigame, crafting, and Options.
6. Verify that changing the new interface locale leaves story and TTS language
   values unchanged.

## Required checks

Run focused tests while editing, followed by the repository gates that cover
the changed layers:

```bash
go test ./...
go vet ./...
cargo fmt --manifest-path gateway/Cargo.toml -- --check
cargo test --manifest-path gateway/Cargo.toml
cargo clippy --manifest-path gateway/Cargo.toml --all-targets -- \
  -A clippy::too_many_arguments -D warnings
cd gateway/web
bun run test
bun run build
bun run test:e2e
cd ../..
make verify
make friend-safe-check
make release-check
make docs-install
make docs-build
bun scripts/check-docs.ts
```

For browser changes, test both desktop and mobile projects with keyboard
navigation, visible focus, reduced motion, screen-reader names, and long Italian
labels. Catalog parity and important-surface literal guards are release checks,
not optional cleanup.

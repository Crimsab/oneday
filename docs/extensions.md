# Extensions and story packs

Story packs add reusable rules and presentation direction without forking the
engine. The tracked example is `plugins/examples/noir-investigation.yaml`.

## What a story pack can define

- identity, genre, description, and recommended RAG model;
- vitals, attributes, secondary stats, currency, and whether combat exists;
- world rules for clocks, weather, travel, and off-screen events;
- difficulty profiles and consequence budgets;
- challenge pools selected by narrative tags and cooldowns;
- minigame definitions with registered kinds, prompts, difficulty, options, and
  accepted answers where required;
- outcome policy, visual bible, negative direction, palette, and voice bible.

## Minimal workflow

Start from the example, give every pack and definition a stable unique ID, and
keep mechanics explicit enough for deterministic validation. Discover and
validate tracked packs with:

```bash
go run ./cmd/oneday story-packs list
```

The command validates each discovered YAML file and exits non-zero for an
invalid pack. CI runs it as part of the release gate.

## Challenge definitions

Registered minigame kinds include `deduction`, `negotiation`, `pattern`,
`bidding`, `courtroom`, `comedy`, `riddle`, `memory`, `rps`, and `quicktime`.
Use narrative tags to describe when a pool fits, and cooldown turns to prevent
repetition. Definitions that depend on a correct response must include explicit
accepted answers; difficulty must remain within the supported range.

Difficulty profiles may require timing-free interactions. Treat that policy as
a compatibility contract: a pool that cannot satisfy it should not be selected.

## Compatibility and safety

Story packs are data, not executable plugins. They cannot introduce arbitrary
code or bypass engine validation. Keep stable IDs once players may have saved
state, validate migrations against existing stories, and avoid placing secrets
or private deployment details in a public pack.

Engine contributors adding a new mechanic must update the Go host, typed
Go/Rust contracts, browser renderer, persistence tests, and compatibility gates
together. See [Development](development.md) and [Testing](testing.md).
